package conf

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
)

func RunServer(router *chi.Mux) {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Kanal für Fehler vom Server
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("HTTP server listening on %s\n", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// Kanal für OS Signale (SIGINT, SIGTERM)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Blockieren bis ein Fehler auftritt oder ein Signal kommt
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Kritischer Serverfehler: %v", err)
		}

	case sig := <-stop:
		log.Printf("Shutdown signal empfangen (%v) — fahre herunter...\n", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Fehler beim Shutdown: %v\n", err)
			srv.Close() // Harter Abbruch, wenn Shutdown klemmt
		}
	}

	log.Println("Server beendet ✅")
}
