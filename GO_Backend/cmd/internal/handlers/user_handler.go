package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"LickLib/cmd/internal/service"

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
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.service.GetUserByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// GET /users/by-username/{username}
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
	//userIDStr := r.FormValue("user_id")
	// TODO: wahrscheinlich nötig hier die ID zu nutzen...
	// 2. Parsen statt konvertieren
	// uuid.Parse prüft auch direkt, ob der String das richtige Format hat
	// (z.B. 8-4-4-4-12 Zeichen)
	/*
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			// Wenn die ID "1" oder "hallo" ist, wird das hier abgefangen
			http.Error(w, "Ungültige User-ID (kein UUID-Format)", http.StatusBadRequest)
			return
		}
	*/

	var emailPtr *string
	mail := r.FormValue("email")

	if mail != "" {
		emailPtr = &mail
	}

	metadata := service.UserMetadata{
		Username: r.FormValue("username"),
		Email:    emailPtr, // Ist nil, wenn mail leer war -> NULL in der DB
	}

	// 1. Validierung (Optional, aber empfohlen)
	if metadata.Username == "" {
		http.Error(w, "Username darf nicht leer sein", http.StatusBadRequest)
		return
	}

	// 2. Den Service aufrufen
	// Wir nutzen r.Context(), um den Request abbrechen zu können, falls der User die Verbindung trennt
	err := h.writeService.CreateUser(r.Context(), metadata)
	if err != nil {
		// Hier könntest du prüfen, ob z.B. der Username schon vergeben ist
		log.Printf("Fehler beim Erstellen des Users: %v", err)
		http.Error(w, "Konnte User nicht erstellen", http.StatusInternalServerError)
		return
	}

	// 3. Erfolgsmeldung zurückgeben
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created

	// Dem Client die neue ID und Infos zurückschicken
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "User erfolgreich erstellt",
	})
}
