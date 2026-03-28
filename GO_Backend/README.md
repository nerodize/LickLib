# LickLib — Backend

> "TikTok für Musiker, aber weniger gehirnzermürbend."  
> Nutzer laden Licks, Riffs und Stücke hoch — mit automatischer Notation, Genre-Tags und algorithmusbasierter Discovery.

---

## Inhaltsverzeichnis

- [Tech Stack](#tech-stack)
- [Architektur](#architektur)
- [Projektstruktur](#projektstruktur)
- [Voraussetzungen](#voraussetzungen)
- [Setup & Startup](#setup--startup)
- [Konfiguration](#konfiguration)
- [API-Übersicht](#api-übersicht)
- [Datenbank & Migrationen](#datenbank--migrationen)
- [Tests](#tests)
- [Docker](#docker)

---

## Tech Stack

| Komponente   | Technologie                          |
|--------------|--------------------------------------|
| Backend      | Go 1.25, Chi Router                  |
| Datenbank    | PostgreSQL 16, GORM, golang-migrate  |
| Object Store | MinIO (S3-kompatibel)                |
| Auth         | Keycloak 24 (OIDC / JWT)             |
| Container    | Docker, Docker Compose               |

---

## Architektur

Das Projekt folgt einer klassischen **Layered Architecture** (Handler → Service → Repository):

```
HTTP Request
     │
     ▼
┌─────────────┐
│   Handler   │  Parsing, Validierung auf HTTP-Ebene, Auth-Context lesen
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Service   │  Business-Logik, Orchestrierung, Validierung auf Domänen-Ebene
└──────┬──────┘
       │
  ┌────┴────┐
  ▼         ▼
┌────┐  ┌─────────┐
│ DB │  │ Storage │   Repository (GORM/Postgres) + MinIO
└────┘  └─────────┘
```

### Authentifizierungs-Flow

```
Client                    Backend                   Keycloak
  │                          │                          │
  │── POST /auth/login ──────▶                          │
  │                          │── Token-Request (ROPC) ──▶
  │                          │◀── JWT ──────────────────│
  │◀── JWT ──────────────────│                          │
  │                          │                          │
  │── POST /tracks ──────────▶                          │
  │   (Bearer JWT)           │── JWKS Verify (lokal) ───▶
  │                          │   (Keys gecacht)         │
  │◀── 201 Created ──────────│                          │
```

### Upload-Flow (DB-First Pattern)

```
1. Metadaten validieren
2. Alte FAILED-Einträge mit gleichem Titel bereinigen
3. Track-Eintrag mit Status = UPLOADING in DB anlegen  ← DB-First
4. Datei zu MinIO hochladen
   ├── Fehler → Status = FAILED setzen (Rollback)
   └── Erfolg → Status = READY + storage_key setzen
```

**Warum DB-First?** Der Track existiert in der DB bevor die Datei in MinIO landet. So können verwaiste Uploads (Crash mid-upload) über den `FAILED`-Status identifiziert und bereinigt werden.

### TrackStatus-Enum

| Status       | Bedeutung                                    |
|--------------|----------------------------------------------|
| `UPLOADING`  | DB-Eintrag angelegt, MinIO-Upload läuft noch |
| `READY`      | Upload abgeschlossen, Track öffentlich        |
| `FAILED`     | Upload fehlgeschlagen (Rollback-Marker)       |
| `PROCESSING` | Reserviert für zukünftige Audio-Verarbeitung  |

---

## Projektstruktur

```
GO_Backend/
├── cmd/
│   ├── api/
│   │   ├── main.go               # Einstiegspunkt
│   │   ├── conf/
│   │   │   ├── database.go       # DB-Setup & Migrations
│   │   │   ├── routes.go         # Dependency Injection & Routing
│   │   │   └── server.go         # HTTP-Server + Graceful Shutdown
│   │   └── middleware/
│   │       └── auth_middleware.go # JWT-Auth & Auth-Simulation (Dev)
│   └── internal/
│       ├── config/
│       │   ├── Config.go         # Konfigurationsstrukturen + Loader
│       │   └── config.yaml       # Lokale Konfiguration (nicht einchecken!)
│       ├── db/                   # DB-Utilities (Open, Seed, Migrate)
│       ├── entity/               # GORM-Modelle (User, Track, Notation)
│       ├── handlers/             # HTTP-Handler (Auth, Track, User)
│       ├── repository/
│       │   ├── track_repository.go   # Interface
│       │   ├── user_repository.go    # Interface
│       │   └── pg/               # PostgreSQL-Implementierungen
│       └── service/
│           ├── track_read_service.go
│           ├── track_write_service.go
│           ├── user_read_service.go
│           └── user_write_service.go
├── cmd/storage/
│   └── minio.go                  # MinIO-Client
├── migrations/                   # SQL-Migrationsdateien (golang-migrate)
├── .extras/docker/
│   ├── dev/docker-compose.yml    # Dev-Stack (Postgres, MinIO, Keycloak)
│   ├── test/docker-compose.yml   # Test-Stack (nur Postgres, in-memory)
│   └── prod/docker-compose.yml   # Prod-Stack (Backend + alle Services)
├── Dockerfile                    # Multi-Stage Build
├── Makefile                      # Shortcuts für alle häufigen Tasks
└── config.yaml                   # App-Konfiguration
```

---

## Voraussetzungen

- **Go** 1.25+
- **Docker** & **Docker Compose**
- **Make** (optional, aber empfohlen)
- **golang-migrate CLI** (für manuelle Migrations-Befehle)

```bash
# golang-migrate installieren (macOS/Linux)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

## Setup & Startup

### 1. Dev-Umgebung starten (Postgres, MinIO, Keycloak)

```bash
make dev
```

Startet alle Infrastruktur-Services im Hintergrund:

| Service  | URL / Port                              |
|----------|-----------------------------------------|
| Postgres | `localhost:5432` (user: postgres, db: licks) |
| MinIO    | `http://localhost:9000`                 |
| MinIO UI | `http://localhost:9001`                 |
| Keycloak | `http://localhost:8081`                 |

### 2. Keycloak einrichten

Beim ersten Start muss Keycloak manuell konfiguriert werden:

1. Öffne `http://localhost:8081` → Login: `admin` / `admin`
2. Erstelle einen neuen Realm: **`licklib`**
3. Erstelle einen Client: **`licklib-backend`**
   - Client Authentication: **ON**
   - Service Account Roles: **ON** (für Admin-Token)
   - Valid Redirect URIs: `*` (Dev)
4. Kopiere den **Client Secret** aus dem Tab "Credentials"
5. Trage ihn in `cmd/internal/config/config.yaml` ein

> **Tipp:** Im Dev-Modus (`app_mode: "dev"`) wird statt JWT-Validierung die `X-User-ID`-Header-Simulation verwendet — Keycloak ist dann für den Upload-Flow optional.

### 3. MinIO-Bucket anlegen

Nach dem ersten Start einmalig:

1. Öffne `http://localhost:9001` → Login: `minio_admin_user` / `supersecretpassword123`
2. Erstelle einen Bucket mit dem Namen **`tracks`**

### 4. Konfiguration prüfen

```bash
# config.yaml liegt im Projekt-Root ODER unter cmd/internal/config/config.yaml
cat config.yaml
```

Beispiel:

```yaml
app_mode: "dev"   # "dev" = Auth-Simulation, "prod" = echtes JWT

bucket:
  endpoint: "localhost:9000"
  access_key: "minio_admin_user"
  secret_key: "supersecretpassword123"
  name: "tracks"

keycloak:
  url: "http://localhost:8081"
  realm: "licklib"
  client_id: "licklib-backend"
  client_secret: "<dein-secret-hier>"
```

### 5. Backend starten

```bash
go run cmd/api/main.go
```

Beim Start werden automatisch alle ausstehenden Migrationen angewendet.

Optional: Mit Seed-Daten starten:

```bash
SEED=true go run cmd/api/main.go
```

---

## Konfiguration

Das Backend lädt seine Konfiguration aus `config.yaml` (Projekt-Root oder `cmd/internal/config/config.yaml`). Umgebungsvariablen überschreiben YAML-Werte (wichtig für Docker/Prod).

| YAML-Pfad              | Env-Variable        | Standard               |
|------------------------|---------------------|------------------------|
| `app_mode`             | `APP_MODE`          | `dev`                  |
| `bucket.endpoint`      | `BUCKET_ENDPOINT`   | `localhost:9000`       |
| `bucket.access_key`    | `BUCKET_ACCESS_KEY` | —                      |
| `bucket.secret_key`    | `BUCKET_SECRET_KEY` | —                      |
| `bucket.name`          | `BUCKET_NAME`       | —                      |
| `keycloak.url`         | `KEYCLOAK_URL`      | `http://localhost:8081`|
| `keycloak.realm`       | `KEYCLOAK_REALM`    | `licklib`              |
| `keycloak.client_id`   | `KEYCLOAK_CLIENT_ID`| `licklib-backend`      |
| DB-DSN                 | `DB_DSN`            | `postgres://postgres:postgres@localhost:5432/licks?sslmode=disable` |

### App-Modi

| Modus  | Auth-Verhalten                                      |
|--------|-----------------------------------------------------|
| `dev`  | `X-User-ID`-Header wird als User-ID akzeptiert (kein JWT) |
| `prod` | JWT wird gegen Keycloak JWKS validiert              |

---

## API-Übersicht

### Öffentliche Endpoints (kein Auth nötig)

| Methode | Pfad                        | Beschreibung              |
|---------|-----------------------------|---------------------------|
| `GET`   | `/tracks/{id}`              | Track per ID abrufen      |
| `GET`   | `/tracks/user/{username}`   | Alle Tracks eines Users   |
| `GET`   | `/tracks/{id}/play`         | Presigned MinIO-URL (Redirect) |
| `GET`   | `/users/{id}`               | User per ID abrufen       |
| `GET`   | `/users/search/{username}`  | User per Username suchen  |
| `POST`  | `/users`                    | User registrieren         |
| `POST`  | `/auth/login`               | Login → JWT               |

### Geschützte Endpoints (Auth erforderlich)

| Methode  | Pfad             | Beschreibung           |
|----------|------------------|------------------------|
| `POST`   | `/tracks`        | Track hochladen (Multipart) |
| `DELETE` | `/tracks/{id}`   | Track löschen (nur Eigentümer) |
| `PATCH`  | `/tracks/{id}`   | Track-Metadaten updaten |
| `DELETE` | `/users/{id}`    | Account löschen (nur selbst) |
| `PATCH`  | `/users/{id}`    | Account updaten        |

#### Track-Upload (Multipart-Form)

```
POST /tracks
Content-Type: multipart/form-data

Felder:
  trackFile    (file)    Audiodatei — MP3, WAV oder FLAC, max. 100 MB
  title        (string)  3–200 Zeichen
  description  (string)  10–2000 Zeichen
  difficulty   (string)  EASY | MEDIUM | HARD | GOGGINS (optional)
```

---

## Datenbank & Migrationen

Migrationen liegen unter `migrations/` und werden beim Start automatisch via `golang-migrate` ausgeführt. Die Reihenfolge ergibt sich aus dem Dateinamen-Präfix.

```
migrations/
├── 001_initial_schema.sql              # Initiales Schema (legacy)
├── 000002_updated_schema.up.sql        # UUID-basiertes Schema
├── 000003_updated_minio_schema.up.sql  # storage_key-Spalte
├── 000004_drop_pw_hash_schema.up.sql   # Passwort-Hash entfernt (→ Keycloak)
├── 000005_add_status_schema.up.sql     # TrackStatus-Enum
└── 000006_fix_unique_constraint.up.sql # Partial Unique Index (nur READY)
```

#### Wichtiger Unique Constraint

```sql
-- Nur READY-Tracks haben einen Unique-Constraint auf (user_id, title).
-- FAILED/UPLOADING-Einträge mit demselben Titel sind erlaubt (Retry-Logik).
CREATE UNIQUE INDEX idx_userid_title_ready
ON tracks (user_id, title)
WHERE status = 'READY';
```

#### Manuelle Migrations-Befehle

```bash
make migrate-up      # Dev-DB hochmigrieren
make migrate-down    # Dev-DB zurückrollen
make migrate-test    # Test-DB hochmigrieren
```

---

## Tests

```bash
make test             # Alle Tests (startet Test-DB automatisch)
make test-unit        # Nur Unit-Tests (kein DB nötig, schnell)
make test-integration # Integration-Tests (mit DB)
make test-coverage    # Coverage-Report → coverage.html
```

### Testaufbau

**Unit-Tests** (`service/`) nutzen Interface-basierte Mocks für Repo und Storage — kein Docker nötig.

**Integration-Tests** (`repository/pg/`) starten eine echte Postgres-Instanz auf Port `5434` (Test-Docker-Compose), führen Migrationen aus und testen die echten GORM-Queries.

```bash
# Test-DB manuell starten
cd .extras/docker/test && docker compose up -d
```

---

## Docker

```bash
# Dev-Stack (Infrastruktur)
make dev          # starten
make dev-down     # stoppen

# Prod-Stack (Backend + alle Services)
make prod-start   # bauen + starten
make prod-stop    # stoppen
make docker-logs  # Backend-Logs verfolgen

# Aufräumen
make clean        # alle Docker-Ressourcen entfernen
```

### Multi-Stage Dockerfile

Das `Dockerfile` baut in zwei Stages:

1. **Builder** (`golang:1.25-alpine`): Kompiliert das Binary mit `CGO_ENABLED=0`
2. **Runtime** (`alpine:latest`): Minimales Image, läuft als Non-Root-User

```bash
docker build -t licklib:latest .
```