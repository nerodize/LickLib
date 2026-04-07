package main

import (
	"LickLib/cmd/api/conf"
	"LickLib/cmd/internal/config"
	"LickLib/cmd/internal/db"
	"LickLib/cmd/internal/metrics"

	"LickLib/cmd/storage"
	"os"

	_ "github.com/lib/pq"
)

// @title           LickLib API
// @version         1.0
// @description     Social Media for musicians — Backend API
// @host            api.188.245.33.223.nip.io
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// 1. Config laden
	cfg := config.LoadConfig("config.yaml")

	metrics.Init()

	// 2. Datenbank & Migrationen
	gdb, sqlDB := conf.SetupDatabase()
	defer sqlDB.Close()
	conf.RunMigrations(sqlDB) // Nutzt jetzt golang-migrate

	if os.Getenv("SEED") == "true" {
		db.Seed(gdb)
	}

	// 3. Router & Dependency Injection
	// Wir übergeben alles Nötige an die setupRoutes Funktion
	minioClient := storage.NewMinioClient(cfg.Bucket)
	router := conf.SetupRoutes(gdb, minioClient, cfg)

	// 4. Start!
	conf.RunServer(router)
}
