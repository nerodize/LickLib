package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"LickLib/cmd/internal/service"

	"github.com/go-chi/chi/v5"
)

type TrackHandler struct {
	service *service.TrackReadService
}

func NewTrackHandler(s *service.TrackReadService) *TrackHandler {
	return &TrackHandler{service: s}
}

func (h *TrackHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid track id", http.StatusBadRequest)
		return
	}

	track, err := h.service.GetTrackByID(uint(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)
}

func (h *TrackHandler) GetByUsername(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	track, err := h.service.GetTracksByUsername(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(track)

}
