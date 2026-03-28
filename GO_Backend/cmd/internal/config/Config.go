package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	// cleanenv wird dann später ziemlich wichtig @Docker und hosting etc.
	"github.com/ilyakaznacheev/cleanenv"
)

// in dieser File ist es ein wenig unaufgeräumt
// TODO: Diese Geschichte noch sorgfältig aufräumen wenn wirklich nötig.
type AppMode string

const (
	Dev  AppMode = "dev"
	Prod AppMode = "prod"
	Test AppMode = "test"
)

type Config struct {
	Bucket   BucketConfig   `yaml:"bucket"`
	Keycloak KeycloakConfig `yaml:"keycloak"`

	Mode AppMode `yaml:"app_mode" env:"APP_MODE" env-default:"dev"`
}

type BucketConfig struct {
	Endpoint  string `yaml:"endpoint" env:"BUCKET_ENDPOINT" env-default:"localhost:9000"`
	AccessKey string `yaml:"access_key" env:"BUCKET_ACCESS_KEY"` // env aktuell unused, Werte direkt gesetzt
	SecretKey string `yaml:"secret_key" env:"BUCKET_SECRET_KEY"`
	Name      string `yaml:"name" env:"BUCKET_NAME"`
}

type KeycloakConfig struct {
	URL          string `yaml:"url"           env:"KEYCLOAK_URL"           env-default:"http://localhost:8081"`
	Realm        string `yaml:"realm"         env:"KEYCLOAK_REALM"         env-default:"licklib"`
	ClientID     string `yaml:"client_id"     env:"KEYCLOAK_CLIENT_ID"     env-default:"licklib-backend"`
	ClientSecret string `yaml:"client_secret"`
}

// Methoden direkt darunter, gehören zum Typ
func (k KeycloakConfig) JWKSUrl() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", k.URL, k.Realm)
}

// für admin auth url auch valid
func (k KeycloakConfig) TokenUrl() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", k.URL, k.Realm)
}

// 2. URL um die User-Liste zu verwalten (POST = Erstellen, GET = Suchen)
func (k KeycloakConfig) AdminUsersUrl() string {
	return fmt.Sprintf("%s/admin/realms/%s/users", k.URL, k.Realm)
}

func LoadConfig(path string) *Config {
	var cfg Config

	// Check, ob die Datei existiert
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Warnung: %s nicht gefunden, versuche alternativen Pfad...", path)
		// Alternative: Suche im Unterordner, falls man aus dem Root startet
		path = "cmd/internal/config/config.yaml"
	}

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		// Hier geben wir den absoluten Pfad aus, damit du im Terminal siehst, wo er gesucht hat
		abs, _ := filepath.Abs(path)
		log.Fatalf("Konfiguration konnte nicht geladen werden unter: %s | Fehler: %v", abs, err)
	}

	return &cfg
}
