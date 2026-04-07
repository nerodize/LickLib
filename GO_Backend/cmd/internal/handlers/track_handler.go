package handlers

import (
	"LickLib/cmd/api/middleware"
	"LickLib/cmd/internal/service"
	"encoding/json"
	"errors"
	"log"
	"mime/multipart"
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

// @Summary      Track per ID abrufen
// @Tags         tracks
// @Produce      json
// @Param        id   path      string  true  "Track UUID"
// @Success      200  {object}  models.Track
// @Failure      400
// @Failure      404
// @Router       /tracks/{id} [get]
func (h *TrackHandler) GetByID(w http.ResponseWriter, r *http.Request) {

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

// @Summary      Alle Tracks eines Users abrufen
// @Tags         tracks
// @Produce      json
// @Param        username  path      string  true  "Username"
// @Success      200       {array}   models.Track
// @Failure      404
// @Router       /tracks/user/{username} [get]
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

// @Summary      Track hochladen
// @Tags         tracks
// @Accept       multipart/form-data
// @Produce      plain
// @Security     BearerAuth
// @Param        trackFile  formData  file    true  "Audiodatei (MP3/WAV/FLAC, max 100MB)"
// @Param        title      formData  string  true  "Titel (3-200 Zeichen)"
// @Param        description formData string  true  "Beschreibung (10-2000 Zeichen)"
// @Param        difficulty formData  string  false "EASY|MEDIUM|HARD|GOGGINS"
// @Success      201
// @Failure      400
// @Failure      401
// @Router       /tracks [post]
func (h *TrackHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	file, header, err := h.parseUploadRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	defer file.Close()

	if err := h.validateUploadFile(header); err != nil {
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
	}

	metadata := h.extractMetadata(r, userID, header)

	err = h.writeService.UploadTrack(r.Context(), file, header.Size, metadata)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Track erfolgreich hochgeladen"))
}

// @Summary      Track löschen
// @Tags         tracks
// @Security     BearerAuth
// @Param        id   path      string  true  "Track UUID"
// @Success      204
// @Failure      401
// @Failure      403
// @Router       /tracks/{id} [delete]
func (h *TrackHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := uuid.Parse(idStr)

	currentUserID := middleware.GetUserID(r.Context())

	// more or less redundant
	if currentUserID == uuid.Nil {
		http.Error(w, "Nicht autorisiert", http.StatusUnauthorized)
		return
	}

	log.Printf("Löschversuch: User %v möchte Track %v löschen", currentUserID, trackID)

	err := h.writeService.DeleteTrack(r.Context(), trackID, currentUserID)
	if err != nil {
		http.Error(w, "Forbidden: "+err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary      Track-Metadaten updaten
// @Tags         tracks
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Track UUID"
// @Param        body body      service.UpdateTrackRequest  true  "Update-Felder"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Router       /tracks/{id} [patch]
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

// @Summary      Track abspielen (Presigned URL)
// @Tags         tracks
// @Param        id   path      string  true  "Track UUID"
// @Success      307
// @Failure      404
// @Router       /tracks/{id}/play [get]
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

// ===== HELPER-FUNKTIONEN =====

func (h *TrackHandler) parseUploadRequest(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	// RAM-Limit
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, nil, errors.New("request too large or faulty")
	}

	// File extrahieren
	file, header, err := r.FormFile("trackFile")
	if err != nil {
		return nil, nil, errors.New("file 'trackFile' is missing")
	}

	return file, header, nil
}

func (h *TrackHandler) validateUploadFile(header *multipart.FileHeader) error {
	// Size-Check
	const maxFileSize = 100 << 20 // 100MB
	if header.Size > maxFileSize {
		return errors.New("file exceeds 100MB limit")
	}

	// Extension-Check
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".mp3" && ext != ".wav" && ext != ".flac" {
		return errors.New("only MP3/WAV/FLAC allowed")
	}

	return nil
}

func (h *TrackHandler) extractMetadata(r *http.Request, userID uuid.UUID, header *multipart.FileHeader) service.TrackMetadata {
	return service.TrackMetadata{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		UserID:      userID,
		Difficulty:  r.FormValue("difficulty"),
		FileExt:     filepath.Ext(header.Filename),
	}
}

func (h *TrackHandler) handleUploadError(w http.ResponseWriter, err error) {
	// UNIQUE constraint error
	if strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") {
		http.Error(w, "Track with this title already exists", http.StatusConflict)
		return
	}

	// Generic error
	log.Printf("Upload Fehler: %v", err)
	http.Error(w, "Fehler beim Speichern des Tracks", http.StatusInternalServerError)
}
