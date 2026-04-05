package conf

import (
	"database/sql"
	"log"
	"os"

	// Die Core-Library
	"github.com/golang-migrate/migrate/v4"
	post "gorm.io/driver/postgres"
	"gorm.io/gorm"

	// Der Treiber für Postgres
	"github.com/golang-migrate/migrate/v4/database/postgres"
	// Der Treiber für das Filesystem
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"LickLib/cmd/internal/helpers"

	_ "github.com/lib/pq"
)

func RunMigrations(sqlDB *sql.DB) {
	// 1. Erstelle den golang-migrate Treiber für Postgres
	// Wichtig: Hier nutzt du das "postgres" Package von migrate, nicht gorm!
	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	helpers.Must0(err)

	// 2. Initialisiere die Migration-Instanz
	// "file://migrations" verweist auf deinen Ordner im Projekt-Root
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	helpers.Must0(err)

	// 3. Migrationen ausführen
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Migration fehlgeschlagen: %v", err)
	}

	log.Println("Migrations ok ✅")
}

func SetupDatabase() (*gorm.DB, *sql.DB) {
	dsn := os.Getenv("DB_DSN")
	log.Println(dsn)
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/licks?sslmode=disable"
	}

	gdb := helpers.Must(gorm.Open(post.Open(dsn), &gorm.Config{}))
	sqlDB := helpers.Must(gdb.DB())
	// sicherstellen, dass DB erreichbar ist
	helpers.Must0(sqlDB.Ping())
	log.Println("DB ok ✅")

	return gdb, sqlDB
}
