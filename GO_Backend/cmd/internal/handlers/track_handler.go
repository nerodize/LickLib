package handlers

import (
	"LickLib/cmd/api/middleware"
	"LickLib/cmd/internal/service"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	// chi for routing
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TrackHandler struct {
	readService  *service.TrackReadService
	writeService *service.TrackWriteService
}

func NewTrackHandler(rs *service.TrackReadService, ws *service.TrackWriteService) *TrackHandler {
	return &TrackHandler{
		readService:  rs,
		writeService: ws,
	}
}

func (h *TrackHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid track id", http.StatusBadRequest)
		return
	}

	track, err := h.readService.GetTrackByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

func (h *TrackHandler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	track, err := h.readService.GetTracksByUsername(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

// vllt auslagern ode reuse für Notation
func (h *TrackHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// Alles was größer ist, wird in temporäre Dateien auf der Platte ausgelagert
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Datei zu groß oder fehlerhaft", http.StatusBadRequest)
		return
	}

	// "trackFile" muss der Key im Frontend/Postman sein
	file, header, err := r.FormFile("trackFile")
	if err != nil {
		http.Error(w, "Datei 'trackFile' konnte nicht gelesen werden", http.StatusBadRequest)
		return
	}
	defer file.Close() // Wichtig: Den Stream am Ende des Handlers schließen!

	userIDStr := r.FormValue("user_id")

	// uuid.Parse prüft auch direkt, ob der String das richtige Format hat
	// (z.B. 8-4-4-4-12 Zeichen)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		// Wenn die ID "1" oder "hallo" ist, wird das hier abgefangen
		http.Error(w, "Ungültige User-ID (kein UUID-Format)", http.StatusBadRequest)
		return
	}

	metadata := service.TrackMetadata{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		UserID:      userID,
		Difficulty:  r.FormValue("difficulty"),
		FileExt:     filepath.Ext(header.Filename),
	}

	err = h.writeService.UploadTrack(r.Context(), file, header.Size, metadata)
	if err != nil {
		log.Printf("Upload Fehler: %v", err)
		http.Error(w, "Fehler beim Speichern des Tracks", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Track erfolgreich hochgeladen"))
}

func (h *TrackHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := strconv.Atoi(idStr)

	currentUserID := middleware.GetUserID(r.Context())

	// more or less redundant
	if currentUserID == uuid.Nil {
		http.Error(w, "Nicht autorisiert", http.StatusUnauthorized)
		return
	}

	log.Printf("Löschversuch: User %d möchte Track %d löschen", currentUserID, trackID)

	err := h.writeService.DeleteTrack(r.Context(), uint(trackID), currentUserID)
	if err != nil {
		http.Error(w, "Forbidden: "+err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TrackHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := strconv.Atoi(idStr)

	currentUserID := middleware.GetUserID(r.Context())

	if currentUserID == uuid.Nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
	}

	var req service.UpdateTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := h.writeService.UpdateTrack(r.Context(), uint(trackID), currentUserID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Track successfully updated"})
}

func (h *TrackHandler) HandlePlay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := strconv.Atoi(idStr)

	// pre baked URL
	playURL, err := h.readService.GetPlaybackURL(r.Context(), uint(trackID))
	if err != nil {
		http.Error(w, "Track not found or faulty link", http.StatusNotFound)
		return
	}

	// Redirect zum Player => browser oder eigener player in der App (könnte schwer werden)
	http.Redirect(w, r, playURL, http.StatusTemporaryRedirect)
}
