package middleware

import (
	"context"
	"net/http"

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

func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok == true {
		return id
	}
	return uuid.Nil
}
