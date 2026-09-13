package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/magomedcoder/repo/internal/usecase"
)

type RepositoryHandler struct {
	create *usecase.CreateUseCase
}

func NewRepositoryHandler(create *usecase.CreateUseCase) *RepositoryHandler {
	return &RepositoryHandler{create: create}
}

type createRepoRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *RepositoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.create.Execute(usecase.CreateInput{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     1,
		BasePath:    "data/repos",
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrNameRequired), errors.Is(err, usecase.ErrInvalidName):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, usecase.ErrAlreadyExists):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":   out.ID,
		"name": out.Name,
		"path": out.Path,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
