package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

type ProfileHandler struct {
	profiles *usecase.ProfileUseCase
}

func NewProfileHandler(profiles *usecase.ProfileUseCase) *ProfileHandler {
	return &ProfileHandler{
		profiles: profiles,
	}
}

type updateProfileRequest struct {
	Email string `json:"email"`
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	var viewerID uint
	if user, ok := middleware.UserFromContext(r.Context()); ok {
		viewerID = user.ID
	}

	out, err := h.profiles.GetByUsername(username, viewerID)
	if err != nil {
		writeProfileError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *ProfileHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.profiles.UpdateEmail(user.ID, req.Email)
	if err != nil {
		writeProfileError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *ProfileHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_avatar")
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_avatar")
		return
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	reader := io.MultiReader(bytes.NewReader(buf[:n]), file)

	out, err := h.profiles.SetAvatar(user.ID, reader, contentType, header.Size)
	if err != nil {
		writeProfileError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *ProfileHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	out, err := h.profiles.ClearAvatar(user.ID)
	if err != nil {
		writeProfileError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ProfileHandler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")
	f, contentType, err := h.profiles.OpenAvatar(username)
	if err != nil {
		writeProfileError(w, r, err)
		return
	}
	
	defer f.Close()
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = io.Copy(w, f)
}

func writeProfileError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrUserNotFound), errors.Is(err, usecase.ErrAvatarNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrInvalidAvatar),
		errors.Is(err, usecase.ErrAvatarTooLarge),
		errors.Is(err, usecase.ErrEmailRequired),
		errors.Is(err, usecase.ErrInvalidEmail):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrEmailTaken):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
