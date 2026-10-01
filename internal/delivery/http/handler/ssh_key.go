package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

type SSHKeyHandler struct {
	keys *usecase.SSHKeyUseCase
}

func NewSSHKeyHandler(keys *usecase.SSHKeyUseCase) *SSHKeyHandler {
	return &SSHKeyHandler{keys: keys}
}

type createSSHKeyRequest struct {
	Title     string `json:"title"`
	PublicKey string `json:"public_key"`
}

func (h *SSHKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createSSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	item, err := h.keys.Create(user.ID, req.Title, req.PublicKey)
	if err != nil {
		writeSSHKeyError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *SSHKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.keys.List(user.ID)
	if err != nil {
		writeSSHKeyError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"keys": items})
}

func (h *SSHKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_ssh_key_id")
		return
	}

	if err := h.keys.Delete(user.ID, id); err != nil {
		writeSSHKeyError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeSSHKeyError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrSSHKeyTitleRequired),
		errors.Is(err, usecase.ErrSSHKeyRequired),
		errors.Is(err, usecase.ErrInvalidSSHKey),
		errors.Is(err, usecase.ErrInvalidSSHKeyID):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrSSHKeyExists):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrSSHKeyNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
