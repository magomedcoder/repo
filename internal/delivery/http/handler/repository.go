package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

type RepositoryHandler struct {
	repos *usecase.RepositoryUseCase
}

func NewRepositoryHandler(repos *usecase.RepositoryUseCase) *RepositoryHandler {
	return &RepositoryHandler{repos: repos}
}

type createRepoRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *RepositoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.repos.Create(usecase.CreateRepositoryInput{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     user.ID,
		BasePath:    "data/repos",
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrNameRequired), errors.Is(err, usecase.ErrInvalidName):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, usecase.ErrAlreadyExists):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, usecase.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          out.ID,
		"name":        out.Name,
		"description": out.Description,
	})
}

func (h *RepositoryHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.repos.List(user.ID)
	if err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"repos": items,
	})
}
