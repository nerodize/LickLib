package middleware

import (
	"context"
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
	// JWKS einmalig laden + auto-refresh
	jwks, _ := keyfunc.NewDefault([]string{jwksURL})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Token aus Header extrahieren
			authHeader := r.Header.Get("Authorization")
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Token validieren (lokal!)
			token, err := jwt.Parse(tokenStr, jwks.Keyfunc)
			if err != nil || !token.Valid {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Claims extrahieren
			claims := token.Claims.(jwt.MapClaims)
			userIDStr := claims["sub"].(string) // Keycloak UUID

			// JETZT PARSEN
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				http.Error(w, "Invalid User ID in Token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok { // oder: ok == true
		return id
	}
	return uuid.Nil
}
