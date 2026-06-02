package handlers

import (
	"LickLib/cmd/api/middleware"
	"LickLib/cmd/internal/service"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type NotationHandler struct {
	readService  *service.NotationReadService
	writeService *service.NotationWriteService
}

func NewNotationHandler(rs *service.NotationReadService, ws *service.NotationWriteService) *NotationHandler {
	return &NotationHandler{
		readService:  rs,
		writeService: ws,
	}
}

func (h *NotationHandler) GetByTrackID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid track id", http.StatusBadRequest)
		return
	}

	notation, err := h.readService.GetByTrackID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notation)
}

func (h *NotationHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	trackIDStr := strings.TrimSpace(chi.URLParam(r, "id"))
	trackID, err := uuid.Parse(trackIDStr)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "request too large or faulty", http.StatusBadRequest)
	}

	file, header, err := h.parseUploadRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	defer file.Close()

	if err := h.validateUploadFile(header); err != nil {
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
	}

	metadata := h.extractMetadata(r, userID, trackID, header)

	if err := h.writeService.UploadNotation(r.Context(), file, header.Size, metadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *NotationHandler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id") // notation ID?
	notationID, _ := uuid.Parse(idStr)

	downloadURL, err := h.readService.GetDownloadURL(r.Context(), notationID)
	if err != nil {
		http.Error(w, "Notation not found or broken link", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, downloadURL, http.StatusTemporaryRedirect)
}

// ===== helper functions ======

var allowedExtensions = map[string]bool{
	".pdf": true,
	".xml": true,
	".mxl": true,
	".gp":  true,
	".gpx": true,
	".gp5": true,
	".txt": true,
}

func (h *NotationHandler) validateUploadFile(header *multipart.FileHeader) error {
	const maxFileSize = 5 << 20
	if header.Size > maxFileSize {
		return errors.New("File exceeds 5Mib Limit")
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		return errors.New("file type not allowed, only PDF, MusicXML, Guitar Pro and TXT files allowed")
	}
	return nil
}

func (h *NotationHandler) parseUploadRequest(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		return nil, nil, errors.New("request too large or faulty")
	}

	file, header, err := r.FormFile("notationFile")
	if err != nil {
		return nil, nil, errors.New("file 'notationFile' is missing")
	}

	return file, header, nil
}
func (h *NotationHandler) extractMetadata(r *http.Request, userID uuid.UUID, trackID uuid.UUID, header *multipart.FileHeader) service.NotationMetadata {
	return service.NotationMetadata{
		TrackID:  trackID,
		AuthorID: userID,
		Type:     r.FormValue("type"),
		FileExt:  filepath.Ext(header.Filename),
	}
}
