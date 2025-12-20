package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"LickLib/cmd/internal/service"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	service *service.UserReadService
}

func NewUserHandler(s *service.UserReadService) *UserHandler {
	return &UserHandler{service: s}
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
