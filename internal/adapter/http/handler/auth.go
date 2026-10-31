package handler

import (
	"net/http"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type AuthHandler struct {
	auth              *usecase.AuthUseCase
	secure            bool
	sameSite          http.SameSite
	allowRegistration bool
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		auth:     auth,
		sameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) SetCookiePolicy(secure bool, sameSite http.SameSite) {
	h.secure = secure
	if sameSite != 0 {
		h.sameSite = sameSite
	}
}

func (h *AuthHandler) SetAllowRegistration(allow bool) {
	h.allowRegistration = allow
}

func (h *AuthHandler) Registration(c *httpx.Context) error {
	return httpx.OK(c, registrationResponse{AllowRegistration: h.allowRegistration})
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

func (h *AuthHandler) Register(c *httpx.Context) error {
	req, err := httpx.BindJSON[registerRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.auth.Register(c.Ctx(), usecase.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return mapError(c, err)
	}

	h.setSessionCookie(c.W, out.Token)
	return httpx.Created(c, userResponse(out.User))
}

func (h *AuthHandler) Login(c *httpx.Context) error {
	req, err := httpx.BindJSON[loginRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.auth.Login(c.Ctx(), usecase.LoginInput{
		Login:    req.Login,
		Password: req.Password,
		Key:      middleware.ClientIP(c.R) + "\n" + strings.ToLower(strings.TrimSpace(req.Login)),
	})
	if err != nil {
		return mapError(c, err)
	}

	h.setSessionCookie(c.W, out.Token)

	return httpx.OK(c, userResponse(out.User))
}

func (h *AuthHandler) Password(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[passwordChangeRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	if err := h.auth.ChangePassword(c.Ctx(), usecase.ChangePasswordInput{
		UserID:    user.ID,
		Current:   req.Current,
		Next:      req.Next,
		KeepToken: middleware.SessionToken(c.R),
	}); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *AuthHandler) Logout(c *httpx.Context) error {
	token := middleware.SessionToken(c.R)
	_ = h.auth.Logout(c.Ctx(), token)
	h.clearSessionCookie(c.W)
	return c.NoContent()
}

func (h *AuthHandler) Me(c *httpx.Context) error {
	user, err := h.auth.Me(c.Ctx(), middleware.SessionToken(c.R))
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, userResponse(user))
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: h.sameSite,
		Expires:  time.Now().Add(14 * 24 * time.Hour),
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
