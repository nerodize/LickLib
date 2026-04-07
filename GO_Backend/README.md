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
- [Kubernetes](#kubernetes)
- [Production Deployment](#production-deployment)
- [Observability](#observability)
- [CI/CD](#cicd)

---

## Tech Stack

| Komponente    | Technologie                                        |
|---------------|----------------------------------------------------|
| Backend       | Go 1.25, Chi Router                                |
| Datenbank     | PostgreSQL 16, GORM, golang-migrate                |
| Object Store  | MinIO (S3-kompatibel)                              |
| Auth          | Keycloak 24 (OIDC / JWT)                           |
| Container     | Docker, Docker Compose (dev / test / prod)         |
| Orchestration | Kubernetes (minikube für lokale Entwicklung)       |
| Observability | Prometheus, Grafana, Alert Rules, SLOs             |
| CI/CD         | GitHub Actions (Unit + Integration Tests + Deploy) |
| Hosting       | Hetzner CX22, nginx Reverse Proxy, Let's Encrypt  |

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

### MinIO Presigned URLs

Der Backend-Service nutzt zwei separate MinIO-Clients:

```
InternalClient  (minio:9000)          → Upload, Delete, List
PublicClient    (188.x.x.x:9000)      → Nur Presigned URLs
```

Der Grund: Presigned URLs werden kryptographisch mit dem Hostnamen des Clients signiert. Würde man den internen Hostnamen (`minio:9000`) nachträglich ersetzen, würde die Signaturprüfung fehlschlagen (`SignatureDoesNotMatch`). Der `PublicClient` verbindet sich direkt mit der öffentlichen IP und signiert die URL damit korrekt.

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
│   │   ├── main.go                   # Einstiegspunkt
│   │   ├── conf/
│   │   │   ├── database.go           # DB-Setup & Migrations
│   │   │   ├── routes.go             # Dependency Injection & Routing
│   │   │   └── server.go             # HTTP-Server + Graceful Shutdown
│   │   └── middleware/
│   │       ├── auth_middleware.go    # JWT-Auth & Auth-Simulation (Dev)
│   │       └── metrics_middleware.go # Prometheus HTTP Middleware
│   └── internal/
│       ├── config/Config.go          # Konfigurationsstrukturen + Loader
│       ├── entity/                   # GORM-Modelle (User, Track, Notation)
│       ├── handlers/                 # HTTP-Handler (Auth, Track, User)
│       ├── metrics/metrics.go        # Prometheus Metriken (4 Golden Signals)
│       ├── repository/
│       │   ├── track_repository.go   # Interface
│       │   ├── user_repository.go    # Interface
│       │   └── pg/                   # PostgreSQL-Implementierungen + Tests
│       └── service/                  # Business-Logik + Unit Tests
├── cmd/storage/minio.go              # MinIO dual-client Setup
├── migrations/                       # SQL-Migrationsdateien (golang-migrate)
├── .extras/
│   ├── docker/
│   │   ├── dev/docker-compose.yml    # Dev-Stack
│   │   ├── test/docker-compose.yml   # Test-Stack (nur Postgres, tmpfs)
│   │   └── prod/docker-compose.yml   # Prod-Stack + .env
│   ├── k8s/                          # Kubernetes Manifeste (minikube)
│   │   ├── backend-deployment.yaml
│   │   ├── backend-configmap.yaml
│   │   ├── backend-secret.yaml
│   │   └── postgres-statefulset.yaml
│   ├── keycloak/licklib-realm.json   # Realm-Export (auto-import beim Start)
│   └── prometheus/
│       ├── prometheus.yml
│       ├── alerts.yml                # Alert Rules (4 SLOs)
│       └── SLO.md                    # SLO-Dokumentation
├── .github/workflows/ci.yml          # GitHub Actions CI/CD
├── Dockerfile                        # Multi-Stage Build
└── Makefile
```

---

## Voraussetzungen

- **Go** 1.25+
- **Docker** & **Docker Compose**
- **Make** (optional, aber empfohlen)
- **golang-migrate CLI** (für manuelle Migrations-Befehle)

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

## Setup & Startup

### 1. Dev-Umgebung starten

```bash
make dev
```

| Service    | URL / Port                                        |
|------------|---------------------------------------------------|
| Postgres   | `localhost:5432` (user: postgres, db: licks)      |
| MinIO      | `http://localhost:9000`                           |
| MinIO UI   | `http://localhost:9001`                           |
| Keycloak   | `http://localhost:8081`                           |
| Prometheus | `http://localhost:9090`                           |
| Grafana    | `http://localhost:3000`                           |

### 2. MinIO-Bucket anlegen

Beim ersten Start einmalig in der MinIO UI (`http://localhost:9001`) einen Bucket mit dem Namen **`tracks`** anlegen.

### 3. Konfiguration

```bash
cp cmd/internal/config/config.yaml.example config.yaml
# Keycloak Client Secret eintragen
```

### 4. Backend starten

```bash
go run cmd/api/main.go
```

---

## Konfiguration

Das Backend lädt Konfiguration aus `config.yaml` (Projekt-Root). Umgebungsvariablen überschreiben YAML-Werte — wichtig für Docker und Production.

| Env-Variable            | Standard                    | Beschreibung                         |
|-------------------------|-----------------------------|--------------------------------------|
| `APP_MODE`              | `dev`                       | `dev` = Auth-Simulation, `prod` = JWT |
| `DB_DSN`                | postgres://...localhost...  | PostgreSQL Connection String         |
| `BUCKET_ENDPOINT`       | `localhost:9000`            | MinIO interner Endpoint              |
| `BUCKET_PUBLIC_URL`     | —                           | MinIO öffentliche URL (für Presigned) |
| `BUCKET_ACCESS_KEY`     | —                           | MinIO Access Key                     |
| `BUCKET_SECRET_KEY`     | —                           | MinIO Secret Key                     |
| `BUCKET_NAME`           | —                           | MinIO Bucket Name                    |
| `KEYCLOAK_URL`          | `http://localhost:8081`     | Keycloak Base URL                    |
| `KEYCLOAK_REALM`        | `licklib`                   | Keycloak Realm                       |
| `KEYCLOAK_CLIENT_ID`    | `licklib-backend`           | Keycloak Client ID                   |
| `KEYCLOAK_CLIENT_SECRET`| —                           | Keycloak Client Secret               |

### App-Modi

| Modus  | Auth-Verhalten                                          |
|--------|---------------------------------------------------------|
| `dev`  | `X-User-ID`-Header wird als User-ID akzeptiert (kein JWT) |
| `prod` | JWT wird gegen Keycloak JWKS validiert                  |

---

## API-Übersicht

### Öffentliche Endpoints

| Methode | Pfad                        | Beschreibung                       |
|---------|-----------------------------|------------------------------------|
| `GET`   | `/health`                   | Health Check                       |
| `GET`   | `/metrics`                  | Prometheus Metriken                |
| `GET`   | `/tracks/{id}`              | Track per ID abrufen               |
| `GET`   | `/tracks/user/{username}`   | Alle Tracks eines Users            |
| `GET`   | `/tracks/{id}/play`         | Presigned MinIO-URL (307 Redirect) |
| `GET`   | `/users/{id}`               | User per ID abrufen                |
| `GET`   | `/users/search/{username}`  | User per Username suchen           |
| `POST`  | `/users`                    | User registrieren                  |
| `POST`  | `/auth/login`               | Login → JWT                        |

### Geschützte Endpoints (Bearer JWT erforderlich)

| Methode  | Pfad             | Beschreibung                        |
|----------|------------------|-------------------------------------|
| `POST`   | `/tracks`        | Track hochladen (Multipart)         |
| `DELETE` | `/tracks/{id}`   | Track löschen (nur Eigentümer)      |
| `PATCH`  | `/tracks/{id}`   | Track-Metadaten updaten             |
| `DELETE` | `/users/{id}`    | Account löschen (nur selbst)        |
| `PATCH`  | `/users/{id}`    | Account updaten                     |

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

Migrationen liegen unter `migrations/` und werden beim Start automatisch via `golang-migrate` ausgeführt.

```
migrations/
├── 001_initial_schema.sql
├── 000002_updated_schema.up.sql        # UUID-basiertes Schema
├── 000003_updated_minio_schema.up.sql  # storage_key-Spalte
├── 000004_drop_pw_hash_schema.up.sql   # Passwort-Hash entfernt (→ Keycloak)
├── 000005_add_status_schema.up.sql     # TrackStatus-Enum
├── 000006_fix_unique_constraint.up.sql # Partial Unique Index (nur READY)
└── 000007_add_name_fields.up.sql       # firstName/lastName für User
```

Wichtiger Partial Unique Index — erlaubt mehrere `FAILED`-Einträge mit gleichem Titel (Retry-Logik), aber nur einen `READY`-Eintrag:

```sql
CREATE UNIQUE INDEX idx_userid_title_ready
ON tracks (user_id, title)
WHERE status = 'READY';
```

```bash
make migrate-up      # Dev-DB hochmigrieren
make migrate-down    # Dev-DB zurückrollen
make migrate-test    # Test-DB hochmigrieren
```

---

## Tests

```bash
make test             # Alle Tests (startet Test-DB automatisch)
make test-unit        # Nur Unit-Tests (kein DB nötig)
make test-integration # Integration-Tests (mit DB)
make test-coverage    # Coverage-Report → coverage.html
```

**Unit-Tests** (`service/`) nutzen Interface-basierte Mocks — kein Docker nötig.

**Integration-Tests** (`repository/pg/`) starten eine echte Postgres-Instanz auf Port `5434`, führen Migrationen via `iofs`-Driver aus (Windows-kompatibel) und testen die echten GORM-Queries.

---

## Docker

```bash
make dev          # Dev-Stack starten (Postgres, MinIO, Keycloak, Prometheus, Grafana)
make dev-down     # Dev-Stack stoppen
make prod-start   # Prod-Stack bauen + starten
make prod-stop    # Prod-Stack stoppen
make docker-logs  # Backend-Logs verfolgen
make clean        # Alle Docker-Ressourcen entfernen
```

### Multi-Stage Dockerfile

1. **Builder** (`golang:1.25-alpine`): kompiliert Binary mit `CGO_ENABLED=0`
2. **Runtime** (`alpine:latest`): minimales Image, läuft als Non-Root-User

---

## Kubernetes

Die Kubernetes-Manifeste unter `.extras/k8s/` sind für lokale Entwicklung mit **minikube** gedacht.

```bash
minikube start --driver=docker
eval $(minikube docker-env)      # Windows: minikube docker-env | Invoke-Expression

docker build -t licklib-backend:latest .

kubectl apply -f .extras/k8s/postgres-statefulset.yaml
kubectl apply -f .extras/k8s/backend-configmap.yaml
kubectl apply -f .extras/k8s/backend-secret.yaml
kubectl apply -f .extras/k8s/backend-deployment.yaml

kubectl get pods
kubectl port-forward svc/licklib-backend 8080:8080
```

| Manifest                    | Typ          | Beschreibung                        |
|-----------------------------|--------------|-------------------------------------|
| `backend-deployment.yaml`   | Deployment   | Backend mit Liveness/Readiness Probe |
| `backend-configmap.yaml`    | ConfigMap    | `config.yaml` als K8s-Objekt        |
| `backend-secret.yaml`       | Secret       | Passwörter + Tokens                 |
| `postgres-statefulset.yaml` | StatefulSet  | Postgres mit PVC (1Gi)              |

> **Hinweis:** Kubernetes wird nur für lokales Lernen und Entwicklung genutzt. Production läuft mit Docker Compose auf Hetzner — auf einem einzelnen Server bringt Kubernetes keinen Mehrwert.

---

## Production Deployment

### Infrastruktur

- **Server:** Hetzner CX22 (2 vCPU, 4 GB RAM)
- **OS:** Ubuntu 24.04
- **Reverse Proxy:** nginx mit SSL via Let's Encrypt (certbot)
- **Domains:** `*.188.245.33.223.nip.io`

| Service    | URL                                        |
|------------|--------------------------------------------|
| API        | `https://api.188.245.33.223.nip.io`        |
| Keycloak   | `https://auth.188.245.33.223.nip.io`       |
| Grafana    | `https://grafana.188.245.33.223.nip.io`    |
| MinIO UI   | `https://minio-ui.188.245.33.223.nip.io`   |

### Stack

Production läuft als Docker Compose Stack unter `.extras/docker/prod/`. Alle Secrets liegen in einer `.env`-Datei auf dem Server (nicht im Repository).

```bash
# Auf dem Server
cd /root/LickLib/GO_Backend/.extras/docker/prod
docker compose up -d --build
```

### Keycloak Production Mode

Keycloak läuft mit `start` (nicht `start-dev`) hinter nginx als SSL-terminierendem Reverse Proxy:

```yaml
command: start --import-realm --hostname=auth.188.245.33.223.nip.io --hostname-strict=false --http-enabled=true
environment:
  KC_PROXY_HEADERS: xforwarded
```

Der Realm wird beim Start automatisch aus `.extras/keycloak/licklib-realm.json` importiert.

---

## Observability

### Prometheus Metriken

Vier Metriken für die **Four Golden Signals**:

| Metrik                                  | Typ          | SLI                  |
|-----------------------------------------|--------------|----------------------|
| `licklib_track_uploads_total`           | CounterVec   | Upload-Erfolgsrate   |
| `licklib_track_upload_duration_seconds` | Histogram    | Upload-Latenz p95    |
| `licklib_http_requests_total`           | CounterVec   | HTTP-Fehlerrate      |
| `licklib_http_request_duration_seconds` | HistogramVec | API-Latenz p95       |

### SLOs

| SLO                  | Ziel   | Zeitfenster |
|----------------------|--------|-------------|
| Upload-Erfolgsrate   | ≥ 99%  | 30 Tage     |
| Upload-Latenz p95    | ≤ 5s   | 30 Tage     |
| HTTP-Fehlerrate      | ≤ 1%   | 30 Tage     |
| API-Latenz p95       | ≤ 500ms| 30 Tage     |

Alert Rules mit Burn Rate Alerting sind in `.extras/prometheus/alerts.yml` definiert. Ausführliche SLO-Dokumentation inkl. Error Budget Berechnung und PromQL-Referenz: `.extras/prometheus/SLO.md`.

---

## CI/CD

GitHub Actions Workflow unter `.github/workflows/ci.yml`:

```
Push zu main
  ├── unit      → go test -short
  ├── integration → go test mit Postgres Service
  └── deploy    → SSH auf Hetzner → git pull → docker compose up --build
                  (nur wenn unit + integration grün)
```

Benötigte GitHub Secrets:

| Secret           | Beschreibung                           |
|------------------|----------------------------------------|
| `SERVER_HOST`    | IP des Hetzner Servers                 |
| `SERVER_USER`    | SSH-User (root)                        |
| `SERVER_SSH_KEY` | Privater SSH-Key (ohne Passphrase)     |