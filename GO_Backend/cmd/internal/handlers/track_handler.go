package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"LickLib/cmd/internal/service"
	dto "LickLib/cmd/internal/service/dto"
	"LickLib/cmd/internal/storage"

	"github.com/go-chi/chi/v5"
)

type TrackHandler struct {
	writeService *service.TrackWriteService
	readService  *service.TrackReadService
	store        storage.Storage
	maxUpload    int64
}

func NewTrackHandler(read *service.TrackReadService, write *service.TrackWriteService, store storage.Storage) *TrackHandler {
	return &TrackHandler{
		readService:  read,
		writeService: write,
		store:        store,
		maxUpload:    200 << 20, // 200 MB z.B.
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

func (h *TrackHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload)
	if err := r.ParseMultipartForm(70 << 20); err != nil {
		http.Error(w, "invalid multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	file, fh, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	userIDStr := r.FormValue("user_id")
	userID, _ := strconv.Atoi(userIDStr)
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = fh.Filename
	}
	desc := r.FormValue("description")

	now := time.Now().UTC().Format("20060102")
	filename := strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	dest := filepath.Join("user_"+strconv.Itoa(userID), now, filename)

	size, err := h.store.Save(file, dest)
	if err != nil {
		http.Error(w, "cannot save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	create := dto.TrackDTO{
		UserID:      uint(userID),
		Title:       title,
		Description: desc,
		FileExt:     ext,
		SizeBytes:   size,
		FileURL:     h.store.URL(dest),
	}

	t, err := h.writeService.CreateTrack(r.Context(), create)
	if err != nil {
		http.Error(w, "cannot store metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(t)
}
