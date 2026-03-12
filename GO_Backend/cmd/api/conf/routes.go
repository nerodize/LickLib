package conf

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"gorm.io/gorm"

	"LickLib/cmd/api/middleware"
	"LickLib/cmd/internal/config"
	"LickLib/cmd/internal/handlers"
	"LickLib/cmd/internal/repository/pg"
	"LickLib/cmd/internal/service"
	"LickLib/cmd/storage"
)

func SetupRoutes(gdb *gorm.DB, minio *storage.MinioClient, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()
	//cfg := config.LoadConfig("minioConfig.yaml")

	minioClient := storage.NewMinioClient(cfg.Bucket)
	// Repos & Services hier initialisieren...
	// --- Repos, Services, Handler initialisieren ---
	userRepo := pg.NewUserRepoGorm(gdb)
	trackRepo := pg.NewTrackRepoGorm(gdb)
	userService := service.NewUserService(userRepo)
	userWriteService := service.NewUserWriteService(userRepo, trackRepo, *minioClient, &cfg.Keycloak)
	userHandler := handlers.NewUserHandler(userService, userWriteService)

	trackReadService := service.NewTrackService(trackRepo, minioClient)
	trackWriteService := service.NewTrackWriteService(minioClient, trackRepo)
	trackHandler := handlers.NewTrackHandler(trackReadService, trackWriteService)

	authHandler := handlers.NewAuthHandler(cfg.Keycloak)
	// public routes
	r.Group(func(r chi.Router) {
		r.Get("/tracks/{id}", trackHandler.GetByID)
		r.Get("/tracks/user/{username}", trackHandler.GetByUsername)
		r.Get("/tracks/{id}/play", trackHandler.HandlePlay)
		r.Get("/users/{id}", userHandler.GetByID)
		r.Get("/users/search/{username}", userHandler.GetByUsername)

		// create user hier public, weil hier keine auth nötig
		r.Post("/users", userHandler.CreateUser)
		r.Post("/auth/login", authHandler.Login)

	})

	var authMiddleware func(http.Handler) http.Handler

	switch cfg.Mode {
	case config.Dev:
		authMiddleware = middleware.AuthSimulation
	case config.Prod:
		authMiddleware = middleware.JWTAuth(cfg.Keycloak.JWKSUrl())
	}

	// private routes, gruppiert nach Notwendigkeit von Autorisierung
	r.Group(func(r chi.Router) {
		// tracks
		//r.Use(middleware.AuthSimulation)
		r.Use(authMiddleware)
		r.Post("/tracks", trackHandler.HandleUpload)
		r.Delete("/tracks/{id}", trackHandler.HandleDelete)
		r.Patch("/tracks/{id}", trackHandler.HandleUpdate)

		// users
		r.Delete("/users/{id}", userHandler.HandleDelete)
		r.Patch("/users/{id}", userHandler.HandleUpdate)
	})

	return r
}
