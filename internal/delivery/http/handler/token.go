package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

type TokenHandler struct {
	tokens *usecase.TokenUseCase
}

func NewTokenHandler(tokens *usecase.TokenUseCase) *TokenHandler {
	return &TokenHandler{tokens: tokens}
}

type createTokenRequest struct {
	Name string `json:"name"`
}

func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.tokens.Create(usecase.CreateTokenInput{UserID: user.ID, Name: req.Name})
	if err != nil {
		writeTokenError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *TokenHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.tokens.List(user.ID)
	if err != nil {
		writeTokenError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": items})
}

func (h *TokenHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_token_id")
		return
	}

	if err := h.tokens.Revoke(user.ID, id); err != nil {
		writeTokenError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeTokenError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrTokenNameRequired):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrTokenNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
