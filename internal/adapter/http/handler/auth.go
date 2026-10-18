package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"git.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"git.magomedcoder.ru/repo/internal/domain"
	"git.magomedcoder.ru/repo/internal/usecase"
)

type AuthHandler struct {
	auth     *usecase.AuthUseCase
	secure   bool
	sameSite http.SameSite
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth, sameSite: http.SameSiteLaxMode}
}

func (h *AuthHandler) SetCookiePolicy(secure bool, sameSite http.SameSite) {
	h.secure = secure
	if sameSite != 0 {
		h.sameSite = sameSite
	}
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
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.auth.Register(usecase.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	h.setSessionCookie(w, out.Token)
	writeJSON(w, http.StatusCreated, userResponse(out.User))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.auth.Login(usecase.LoginInput{
		Login:    req.Login,
		Password: req.Password,
		Key:      r.RemoteAddr + "\n" + req.Login,
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	h.setSessionCookie(w, out.Token)
	writeJSON(w, http.StatusOK, userResponse(out.User))
}

func (h *AuthHandler) Password(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	if err := h.auth.ChangePassword(usecase.ChangePasswordInput{
		UserID:  user.ID,
		Current: req.Current,
		Next:    req.Next,
	}); err != nil {
		writeAuthError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := middleware.SessionToken(r)
	_ = h.auth.Logout(token)
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.auth.Me(middleware.SessionToken(r))
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, userResponse(user))
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrUsernameRequired),
		errors.Is(err, usecase.ErrEmailRequired),
		errors.Is(err, usecase.ErrPasswordRequired),
		errors.Is(err, usecase.ErrInvalidUsername),
		errors.Is(err, usecase.ErrInvalidEmail),
		errors.Is(err, usecase.ErrPasswordTooShort),
		errors.Is(err, usecase.ErrReservedSlug):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrUsernameTaken),
		errors.Is(err, usecase.ErrEmailTaken):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrInvalidCredentials),
		errors.Is(err, usecase.ErrUnauthorized),
		errors.Is(err, usecase.ErrSessionExpired):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	case errors.Is(err, usecase.ErrRegistrationClosed):
		writeError(w, r, http.StatusForbidden, err.Error())
	case errors.Is(err, usecase.ErrTooManyAttempts):
		writeError(w, r, http.StatusTooManyRequests, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: h.sameSite,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: h.sameSite,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func userResponse(user *domain.User) map[string]any {
	return map[string]any{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"has_avatar": user.AvatarPath != "",
		"created_at": user.CreatedAt,
	}
}
