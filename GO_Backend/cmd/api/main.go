package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"LickLib/cmd/internal/config"
	"LickLib/cmd/internal/db"
	"LickLib/cmd/internal/handlers"
	"LickLib/cmd/internal/helpers"
	"LickLib/cmd/internal/repository/pg"
	"LickLib/cmd/internal/service"
	"LickLib/cmd/storage"
)

func main() {
	// --- DB öffnen ---
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/licks?sslmode=disable"
	}

	gdb := helpers.Must(gorm.Open(postgres.Open(dsn), &gorm.Config{}))
	sqlDB := helpers.Must(gdb.DB())
	// sicherstellen, dass DB erreichbar ist
	helpers.Must0(sqlDB.Ping())
	log.Println("DB ok ✅")

	// --- Migrations (vor HTTP-Server!) ---
	migrations := []string{
		"migrations/002_updated_schema.sql",
		"migrations/003_updated_minio_schema.sql",
	}

	for _, m := range migrations {
		if _, err := os.Stat(m); err == nil {
			log.Printf("apply migration: %s\n", m)
			helpers.Must0(db.RunSQLFile(sqlDB, m))
		} else {
			log.Printf("Migration %s nicht gefunden, überspringe\n", m)
		}
	}

	cfg := config.LoadConfig("minioConfig.yaml")

	minioClient := storage.NewMinioClient(cfg.Bucket)

	// --- Seed (nur wenn gewollt; in Prod ggf. deaktivieren) ---
	helpers.Must0(db.Seed(gdb))
	log.Println("Migrations & Seed fertig ✅")

	// --- Repos, Services, Handler initialisieren ---
	userRepo := pg.NewUserRepoGorm(gdb)
	userService := service.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userService)

	trackRepo := pg.NewTrackRepoGorm(gdb)
	trackReadService := service.NewTrackService(trackRepo, minioClient)
	trackWriteService := service.NewTrackWriteService(minioClient, trackRepo)
	trackHandler := handlers.NewTrackHandler(trackReadService, trackWriteService)

	r := chi.NewRouter()
	r.Get("/users/{id}", userHandler.GetByID)
	// passt die Route so? @naming conventions
	r.Get("/users/username/{username}", userHandler.GetByUsername)
	r.Get("/tracks/{id}", trackHandler.GetByID)
	r.Get("/tracks/by-username/{username}", trackHandler.GetByUsername)
	r.Get("/{id}/play", trackHandler.HandlePlay)
	r.Post("/tracks/upload", trackHandler.HandleUpload)

	r.Delete("/tracks/delete/{id}", trackHandler.HandleDelete)
	r.Patch("/tracks/update/{id}", trackHandler.HandleUpdate)
	// --- HTTP Server mit Graceful Shutdown ---
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Server in Goroutine starten
	go func() {
		log.Printf("HTTP server listening on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Auf Signal warten (SIGINT, SIGTERM)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop // blockiert bis Signal empfangen

	log.Println("Shutdown signal empfangen — server wird heruntergefahren...")

	// Kontext mit Timeout für Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Fehler beim Shutdown: %v\n", err)
	} else {
		log.Println("HTTP server sauber gestoppt")
	}

	// DB schließen
	if err := sqlDB.Close(); err != nil {
		log.Printf("Fehler beim Schliessen der DB: %v\n", err)
	} else {
		log.Println("DB connection closed")
	}

	log.Println("Beendet ✅")
}
