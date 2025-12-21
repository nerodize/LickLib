package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Bucket BucketConfig `yaml:"bucket"`
}

type BucketConfig struct {
	Endpoint  string `yaml:"endpoint" env:"BUCKET_ENDPOINT" env-default:"localhost:9000"`
	AccessKey string `yaml:"access_key" env:"BUCKET_ACCESS_KEY"`
	SecretKey string `yaml:"secret_key" env:"BUCKET_SECRET_KEY"`
	Name      string `yaml:"name" env:"BUCKET_NAME"`
}

func LoadConfig(path string) *Config {
	var cfg Config

	// Check, ob die Datei existiert
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Warnung: %s nicht gefunden, versuche alternativen Pfad...", path)
		// Alternative: Suche im Unterordner, falls man aus dem Root startet
		path = "cmd/internal/config/minioConfig.yaml"
	}

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		// Hier geben wir den absoluten Pfad aus, damit du im Terminal siehst, wo er gesucht hat
		abs, _ := filepath.Abs(path)
		log.Fatalf("Konfiguration konnte nicht geladen werden unter: %s | Fehler: %v", abs, err)
	}

	return &cfg
}
