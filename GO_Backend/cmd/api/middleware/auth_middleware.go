package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthSimulation(next http.Handler) http.Handler {
	// hier der Adapter als anonyme Funktion => einfachster Weg, also boilerplate
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get("X-User-ID")

		if userIDStr == "" {
			http.Error(w, "No User-ID Header found(Auth Simulation)", http.StatusUnauthorized)
			return
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, "User-ID invalid (must be a valid UUID)", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func JWTAuth(jwksURL string) func(http.Handler) http.Handler {
	// JWKS einmalig laden + auto-refresh im Hintergrund
	// Die Library kümmert sich darum, die Keys von Keycloak zu holen
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Fatalf("Fehler beim Initialisieren des JWKS-Keys: %v", err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Token aus Header extrahieren
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization Header fehlt", http.StatusUnauthorized)
				return
			}

			// Das "Bearer " Präfix entfernen
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader { // Falls kein "Bearer " gefunden wurde
				http.Error(w, "Ungültiges Authorization-Format", http.StatusUnauthorized)
				return
			}

			// 2. Token validieren (Lokal gegen die geladenen Keys)
			token, err := jwt.Parse(tokenStr, jwks.Keyfunc)

			if err != nil {
				// Sehr wichtig für dein Debugging in Bruno!
				fmt.Printf("JWT Validation Error: %v\n", err)
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// 3. Claims prüfen und User-ID extrahieren
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				userIDStr, ok := claims["sub"].(string) // Keycloak nutzt standardmäßig "sub" für die UUID
				if !ok {
					log.Println("Sub claim fehlt im Token")
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				// 4. In UUID Typ parsen
				userID, err := uuid.Parse(userIDStr)
				if err != nil {
					log.Printf("Ungültige User ID im Token: %v", err)
					http.Error(w, "Invalid User ID in Token", http.StatusUnauthorized)
					return
				}

				// 5. UserID in den Context schreiben und weiterreichen
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "Token ungültig", http.StatusUnauthorized)
			}
		})
	}
}

func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok { // oder: ok == true
		return id
	}
	return uuid.Nil
}
