package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"LickLib/cmd/api/middleware"
	"LickLib/cmd/internal/service"

	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	service      *service.UserReadService
	writeService *service.UserWriteService //TODO: hier muss der abhängige code ergänzt werden!!!
}

func NewUserHandler(s *service.UserReadService, ws *service.UserWriteService) *UserHandler {
	return &UserHandler{
		service:      s,
		writeService: ws,
	}
}

// GET /users/{id}
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	user, err := h.service.GetUserByUsername(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// 1. Daten extrahieren (wie gehabt)
	var emailPtr *string
	mail := r.FormValue("email")
	if mail != "" {
		emailPtr = &mail
	}

	metadata := service.UserMetadata{
		Username: r.FormValue("username"),
		Email:    emailPtr,
		Password: r.FormValue("password"),
	}

	// 2. Validierung
	if metadata.Username == "" || metadata.Password == "" {
		http.Error(w, "Username und Passwort sind erforderlich", http.StatusBadRequest)
		return
	}

	// 3. Service aufrufen (gibt jetzt die neue UUID zurück)
	newID, err := h.writeService.CreateUser(r.Context(), metadata)
	if err != nil {
		log.Printf("Fehler beim Erstellen des Users: %v", err)

		// Kleiner Bonus: Spezifischer Fehler für "User existiert bereits"
		if errors.Is(err, service.ErrUserAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict) // 409
			return
		}

		http.Error(w, "Konnte User nicht erstellen", http.StatusInternalServerError)
		return
	}

	// 4. Erfolgsmeldung mit der neuen ID zurückgeben
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"id":      newID, // <--- EXTREM WICHTIG FÜR BRUNO!
		"message": "User erfolgreich in Keycloak und DB erstellt",
	})
}

func (h *UserHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid uuid", http.StatusBadRequest)
		return
	}

	currentUserID := middleware.GetUserID(r.Context())

	// 1. Authentifizierung prüfen
	if currentUserID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// 2. Autorisierung prüfen (Darf er das?)
	// Nur der User selbst (oder später ein Admin) darf löschen
	if currentUserID != userID {
		log.Printf("[SECURITY] User %s tried to delete User %s", currentUserID, userID)
		http.Error(w, "forbidden: you can only delete your own account", http.StatusForbidden)
		return
	}

	log.Printf("Löschvorgang gestartet für User: %s", userID)

	err = h.writeService.DeleteUser(r.Context(), userID)
	if err != nil {
		log.Printf("Fehler beim Löschen: %v", err)
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	log.Printf("DEBUG: URL-ID ist: %s", idStr)

	currentUserID := middleware.GetUserID(r.Context())
	log.Printf("DEBUG: Context-UserID ist: %s", currentUserID)

	if currentUserID == uuid.Nil {
		log.Println("DEBUG: Unauthorized - Context-ID ist Nil")
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	// JSON einlesen
	var req service.UpdateUserRequest
	bodyBytes, _ := io.ReadAll(r.Body) // Body zwischenspeichern zum Loggen
	log.Printf("DEBUG: Body erhalten: %s", string(bodyBytes))

	// Da wir io.ReadAll genutzt haben, müssen wir den Body für den Decoder wieder "füllen"
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("DEBUG: JSON Decode Fehler: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Println("DEBUG: Rufe Service auf...")
	err := h.writeService.UpdateUser(r.Context(), currentUserID, req)
	if err != nil {
		log.Printf("DEBUG: SERVICE ERROR: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("--- DEBUG: Update erfolgreich ---")
	w.WriteHeader(http.StatusOK)
}
