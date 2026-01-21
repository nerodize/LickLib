package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"LickLib/cmd/internal/service"

	// chi for routing
	"github.com/go-chi/chi/v5"
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

func (h *TrackHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 1. Speicherlimit festlegen (z.B. 32 MB)
	// Alles was größer ist, wird in temporäre Dateien auf der Platte ausgelagert
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Datei zu groß oder fehlerhaft", http.StatusBadRequest)
		return
	}

	// 2. Die Datei aus dem Formular fischen
	// "trackFile" muss der Key im Frontend/Postman sein
	file, header, err := r.FormFile("trackFile")
	if err != nil {
		http.Error(w, "Datei 'trackFile' konnte nicht gelesen werden", http.StatusBadRequest)
		return
	}
	defer file.Close() // Wichtig: Den Stream am Ende des Handlers schließen!

	// 3. Metadaten aus den Form-Feldern extrahieren
	// In Go kommen FormValue-Rückgaben immer als String
	userIDStr := r.FormValue("user_id")
	// Kleiner Helfer: String zu Int konvertieren (error handling weggelassen für Kürze)
	userID, _ := strconv.Atoi(userIDStr)

	metadata := service.TrackMetadata{
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		UserID:      userID,
		Difficulty:  r.FormValue("difficulty"),
		FileExt:     filepath.Ext(header.Filename), // Extrahiert .mp3, .wav etc.
	}

	// 4. Den Write-Service aufrufen (Streaming startet hier)
	// Wir geben 'file' (den io.Reader) direkt weiter
	err = h.writeService.UploadTrack(r.Context(), file, header.Size, metadata)
	if err != nil {
		log.Printf("Upload Fehler: %v", err)
		http.Error(w, "Fehler beim Speichern des Tracks", http.StatusInternalServerError)
		return
	}

	// 5. Erfolg melden
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Track erfolgreich hochgeladen"))
}

func (h *TrackHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := strconv.Atoi(idStr)

	debugUserID := r.URL.Query().Get("asUser")

	currentUserID := 1 // Dein Standard-Fallback
	if debugUserID != "" {
		if val, err := strconv.Atoi(debugUserID); err == nil {
			currentUserID = val
		}
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

	// Dein asUser-Hack für Tests, genauso wie bei delete
	currentUserID := 1
	if debugID := r.URL.Query().Get("asUser"); debugID != "" {
		currentUserID, _ = strconv.Atoi(debugID)
	}

	var req service.UpdateTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ungültiges JSON", http.StatusBadRequest)
		return
	}

	err := h.writeService.UpdateTrack(r.Context(), uint(trackID), currentUserID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TrackHandler) HandlePlay(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	trackID, _ := strconv.Atoi(idStr)

	// Der Service liefert uns jetzt direkt die fertige URL
	playURL, err := h.readService.GetPlaybackURL(r.Context(), uint(trackID))
	if err != nil {
		http.Error(w, "Track nicht gefunden oder Link-Fehler", http.StatusNotFound)
		return
	}

	// Redirect zum Player
	http.Redirect(w, r, playURL, http.StatusTemporaryRedirect)
}
