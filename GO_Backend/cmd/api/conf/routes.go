package conf

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	mid "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"gorm.io/gorm"

	"LickLib/cmd/api/middleware"
	"LickLib/cmd/internal/config"
	"LickLib/cmd/internal/handlers"
	"LickLib/cmd/internal/repository/pg"
	"LickLib/cmd/internal/service"
	"LickLib/cmd/storage"

	httpSwagger "github.com/swaggo/http-swagger"

	_ "LickLib/docs" // generierte docs

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRoutes(gdb *gorm.DB, minio *storage.MinioClient, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(mid.Recoverer)
	r.Use(middleware.PrometheusMiddleware) // ← vor allen anderen Middlewares

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

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
		// muss hier stehen sonst unlogisch => hier bekommt man erst "Ausweis"
		r.Post("/auth/login", authHandler.Login)

		// prometheus
		r.Handle("/metrics", promhttp.Handler())

		r.Get("/swagger/*", httpSwagger.WrapHandler)

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
