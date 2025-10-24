package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq" // wichtig für *sql.DB
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	models "LickLib/cmd/internal/entity"
	//"LickLib/src/user/repository"
)

func kek() {

	//var db any
	// Robustere DSN (search_path=public hilft, falls Schema nicht default ist)
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/licks?sslmode=disable&search_path=public"
	}

	// --- 1️⃣ Migration-File ausführen ---
	if err := runInitialSQL(dsn, "migrations/001_initial_schema.sql"); err != nil {
		log.Fatalf("Init-Migration fehlgeschlagen: %v", err)
	}
	log.Println("Initiale SQL-Migration erfolgreich ausgeführt ✅")

	// DB öffnen
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der Datenbank: %v", err)
	}

	// Verbindung testen
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Fehler beim Zugriff auf *sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Datenbank nicht erreichbar: %v", err)
	}
	log.Println("DB-Verbindung erfolgreich hergestellt")

	if err := db.AutoMigrate(&models.User{}, &models.Track{}); err != nil {
		log.Fatalf("AutoMigrate: %v", err)
	}
	log.Println("Tabellen migriert (users, tracks)")

	// --- Seed: robust mit FirstOrCreate ---
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		log.Fatalf("Count(users): %v", err)
	}
	if count == 0 {
		now := time.Now()

		// User anlegen oder holen
		email := "max@example.com"
		u := models.User{
			Username:     "max",
			Email:        &email,
			PasswordHash: "dummy-hash", // TODO: bcrypt/argon2
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := db.Where("username = ?", u.Username).FirstOrCreate(&u).Error; err != nil {
			log.Fatalf("seed user: %v", err)
		}

		// Track anlegen oder holen
		diff := models.DifficultyEasy // aktuell TEXT in deinem Model (kein ENUM)
		t := models.Track{
			Username:    u.Username,
			Title:       "Pentatonic Lick #1",
			Description: "A minor box 1",
			Difficulty:  &diff,
			FileExt:     "wav",
			SizeBytes:   123456,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		// Identifizieren wir den Track minimal über (username, title)
		if err := db.
			Where("username = ? AND title = ?", t.Username, t.Title).
			FirstOrCreate(&t).Error; err != nil {
			log.Fatalf("seed track: %v", err)
		}

		log.Println("Seed-Daten eingefügt ✅ (User: max, 1 Track)")
	} else {
		log.Println("Seed übersprungen (users nicht leer)")
	}
}

func runInitialSQL(dsn string, filePath string) error {
	sqlFile, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(string(sqlFile))
	return err
}
