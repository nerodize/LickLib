package main

import (
	"log"
	"os"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"LickLib/cmd/internal/db"
	models "LickLib/cmd/internal/entity"
)

func main() {
	// --- DB öffnen ---
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/licks?sslmode=disable"
	}
	gdb := must(gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // Dev-safe
	}))
	sqlDB := must(gdb.DB())
	must0(sqlDB.Ping())
	log.Println("DB ok ✅")

	// --- Optional: Initial-SQL ---
	if _, err := os.Stat("migrations/001_initial_schema.sql"); err == nil {
		must0(db.RunSQLFile(sqlDB, "migrations/001_initial_schema.sql"))
	}

	// --- AutoMigrate & Seed ---
	must0(gdb.AutoMigrate(&models.User{}, &models.Track{}))
	must0(db.Seed(gdb))

	log.Println("fertig ✅")
}

// --- helpers ---

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func must0(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
