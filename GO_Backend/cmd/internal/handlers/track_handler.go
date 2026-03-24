package handlers

import (
	"LickLib/cmd/api/middleware"
	"LickLib/cmd/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
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
	fmt.Println("HANDLER FEUERT FÜR ID:", chi.URLParam(r, "id")) // <--- DAS HIER

	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid track id", http.StatusBadRequest)
		return
	}

	track, err := h.readService.GetTrackByID(id)
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
	currentUserID := middleware.GetUserID(r.Context())
	if currentUserID == uuid.Nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "track is too large or faulty", http.StatusBadRequest)
	}

	file, header, err := r.FormFile("trackFile")
	if err != nil {
		http.Error(w, "file: 'trackFile' is missing", http.StatusBadRequest)
	}
	defer file.Close()

	const maxFileSize = 100 << 20
	if header.Size > maxFileSize {
		http.Error(w, "file exeeds: 100MB", 413)
	}

	// ein wenig redundant?
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".mp3" && ext != ".wav" && ext != ".flac" {
		http.Error(w, "only MP3/WAV/FLAC allowed", 400)
		return
	}

	// Die userID kommt jetzt SICHER aus dem Context, nicht vom User-Input!
	metadata := service.TrackMetadata{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		UserID:      currentUserID, // <--- HIER passiert die Magie (middleware)
		Difficulty:  r.FormValue("difficulty"),
		FileExt:     filepath.Ext(header.Filename),
	}

	// 4. Service-Aufruf
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
	trackID, _ := uuid.Parse(idStr)

	currentUserID := middleware.GetUserID(r.Context())

	// more or less redundant
	if currentUserID == uuid.Nil {
		http.Error(w, "Nicht autorisiert", http.StatusUnauthorized)
		return
	}

	log.Printf("Löschversuch: User %d möchte Track %d löschen", currentUserID, trackID)

	err := h.writeService.DeleteTrack(r.Context(), trackID, currentUserID)
	if err != nil {
		http.Error(w, "Forbidden: "+err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TrackHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := uuid.Parse(idStr)

	currentUserID := middleware.GetUserID(r.Context())

	if currentUserID == uuid.Nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
	}

	var req service.UpdateTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := h.writeService.UpdateTrack(r.Context(), trackID, currentUserID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Track successfully updated"})
}

func (h *TrackHandler) HandlePlay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := uuid.Parse(idStr)

	// pre baked URL
	playURL, err := h.readService.GetPlaybackURL(r.Context(), trackID)
	if err != nil {
		http.Error(w, "Track not found or faulty link", http.StatusNotFound)
		return
	}

	// Redirect zum Player => browser oder eigener player in der App (könnte schwer werden)
	http.Redirect(w, r, playURL, http.StatusTemporaryRedirect)
}
