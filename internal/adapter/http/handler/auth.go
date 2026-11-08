package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type AuthHandler struct {
	auth              *usecase.AuthUseCase
	access            *audit.Logger
	secure            bool
	sameSite          http.SameSite
	csrfSecret        string
	allowRegistration bool
}

func NewAuthHandler(auth *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		auth:     auth,
		sameSite: http.SameSiteLaxMode,
	}
}

func (h *AuthHandler) SetAccessLogger(log *audit.Logger) {
	h.access = log
}

func (h *AuthHandler) SetCookiePolicy(secure bool, sameSite http.SameSite) {
	h.secure = secure
	if sameSite != 0 {
		h.sameSite = sameSite
	}
}

func (h *AuthHandler) SetCSRFSecret(secret string) {
	h.csrfSecret = secret
}

func (h *AuthHandler) SetAllowRegistration(allow bool) {
	h.allowRegistration = allow
}

func (h *AuthHandler) Registration(c *httpx.Context) error {
	return httpx.OK(c, registrationResponse{AllowRegistration: h.allowRegistration})
}

func (h *AuthHandler) CSRF(c *httpx.Context) error {
	token := middleware.EnsureCSRFCookie(c.W, c.R, h.secure, h.csrfSecret)
	return httpx.OK(c, map[string]string{"csrf_token": token})
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
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		IP:        middleware.ClientIP(c.R),
		UserAgent: c.R.UserAgent(),
	})
	if err != nil {
		return mapError(c, err)
	}

	h.setSessionCookie(c.W, c.R, out.Token)
	h.rotateSessionCSRF(c.W, c.R, out.Token)
	if h.access != nil {
		h.access.Register(c.R, out.User)
	}
	return httpx.Created(c, userResponse(out.User))
}

func (h *AuthHandler) Login(c *httpx.Context) error {
	req, err := httpx.BindJSON[loginRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.auth.Login(c.Ctx(), usecase.LoginInput{
		Login:     req.Login,
		Password:  req.Password,
		Key:       middleware.ClientIPKey(c.R) + "\n" + strings.ToLower(strings.TrimSpace(req.Login)),
		IP:        middleware.ClientIP(c.R),
		UserAgent: c.R.UserAgent(),
	})
	if err != nil {
		if h.access != nil {
			h.access.FailedAuth(c.R, strings.TrimSpace(req.Login))
		}
		return mapError(c, err)
	}

	h.setSessionCookie(c.W, c.R, out.Token)
	h.rotateSessionCSRF(c.W, c.R, out.Token)
	if h.access != nil {
		h.access.Login(c.R, out.User)
	}

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
		KeepToken: middleware.SessionTokenFor(c.R, h.secure),
	}); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *AuthHandler) Logout(c *httpx.Context) error {
	token := middleware.SessionTokenFor(c.R, h.secure)
	user, _ := h.auth.Me(c.Ctx(), token)
	_ = h.auth.Logout(c.Ctx(), token)

	h.clearSessionCookie(c.W, c.R)
	h.rotateAnonymousCSRF(c.W, c.R)
	if h.access != nil {
		h.access.Logout(c.R, user)
	}

	return c.NoContent()
}

func (h *AuthHandler) Me(c *httpx.Context) error {
	token := middleware.SessionTokenFor(c.R, h.secure)
	if token == "" {
		return c.JSON(http.StatusOK, nil)
	}

	user, err := h.auth.Me(c.Ctx(), token)
	if err != nil {
		if appErr, ok := errors.AsType[*usecase.Error](err); ok && appErr.Kind == usecase.KindUnauthorized {
			h.clearSessionCookie(c.W, c.R)
			h.rotateAnonymousCSRF(c.W, c.R)

			return c.Error(http.StatusUnauthorized, appErr.Code)
		}

		return mapError(c, err)
	}

	return httpx.OK(c, userResponse(user))
}

type sessionsResponse struct {
	Sessions []usecase.SessionItem `json:"sessions"`
}

func (h *AuthHandler) ListSessions(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	items, err := h.auth.ListSessions(c.Ctx(), user.ID, middleware.SessionTokenFor(c.R, h.secure))
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, sessionsResponse{Sessions: items})
}

func (h *AuthHandler) RevokeSession(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	id, err := parseID(c.Param("id"))
	if err != nil || id == 0 {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	revokedCurrent, err := h.auth.RevokeSession(c.Ctx(), user.ID, id, middleware.SessionTokenFor(c.R, h.secure))
	if err != nil {
		return mapError(c, err)
	}

	if revokedCurrent {
		h.clearSessionCookie(c.W, c.R)
		h.rotateAnonymousCSRF(c.W, c.R)
	}

	return c.NoContent()
}

func (h *AuthHandler) RevokeOtherSessions(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	token := middleware.SessionTokenFor(c.R, h.secure)
	if err := h.auth.RevokeOtherSessions(c.Ctx(), user.ID, token); err != nil {
		return mapError(c, err)
	}
	return c.NoContent()
}

func (h *AuthHandler) RevokeAllSessions(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	currentPassword := ""
	if req, bindErr := httpx.BindJSON[stepUpRequest](c); bindErr == nil {
		currentPassword = req.CurrentPassword
	}

	if err := h.auth.RevokeAllSessions(c.Ctx(), user.ID, currentPassword); err != nil {
		return mapError(c, err)
	}
	h.clearSessionCookie(c.W, c.R)
	h.rotateAnonymousCSRF(c.W, c.R)

	return c.NoContent()
}

func (h *AuthHandler) rotateSessionCSRF(w http.ResponseWriter, r *http.Request, sessionToken string) {
	if h.csrfSecret == "" || sessionToken == "" {
		return
	}
	middleware.RotateSessionCSRF(w, r, h.secure, h.csrfSecret, sessionToken)
}

func (h *AuthHandler) rotateAnonymousCSRF(w http.ResponseWriter, r *http.Request) {
	middleware.RotateAnonymousCSRF(w, r, h.secure)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName(r, h.secure),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure || middleware.IsHTTPS(r),
		SameSite: h.sameSite,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	name := middleware.SessionCookieName(r, h.secure)
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure || middleware.IsHTTPS(r) || name == middleware.HostSessionCookie,
		SameSite: h.sameSite,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
