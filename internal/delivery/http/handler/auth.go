package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
)

const sessionCookieName = "session"

type AuthHandler struct {
	auth *usecase.AuthUseCase
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.auth.Register(usecase.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}

	setSessionCookie(w, out.Token)
	writeJSON(w, http.StatusCreated, userResponse(out.User))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.auth.Login(usecase.LoginInput{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}

	setSessionCookie(w, out.Token)
	writeJSON(w, http.StatusOK, userResponse(out.User))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := middleware.SessionToken(r)
	_ = h.auth.Logout(token)
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.Me(middleware.SessionToken(r))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userResponse(user))
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrUsernameRequired),
		errors.Is(err, usecase.ErrEmailRequired),
		errors.Is(err, usecase.ErrPasswordRequired),
		errors.Is(err, usecase.ErrInvalidUsername),
		errors.Is(err, usecase.ErrInvalidEmail),
		errors.Is(err, usecase.ErrPasswordTooShort):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrUsernameTaken),
		errors.Is(err, usecase.ErrEmailTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrInvalidCredentials),
		errors.Is(err, usecase.ErrUnauthorized),
		errors.Is(err, usecase.ErrSessionExpired):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func userResponse(user *domain.User) map[string]any {
	return map[string]any{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	}
}
