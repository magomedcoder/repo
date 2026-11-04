package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type TokenHandler struct {
	tokens *usecase.TokenUseCase
	access *audit.Logger
}

func NewTokenHandler(tokens *usecase.TokenUseCase) *TokenHandler {
	return &TokenHandler{tokens: tokens}
}

func (h *TokenHandler) SetAccessLogger(log *audit.Logger) {
	h.access = log
}

type createTokenRequest struct {
	Name      string  `json:"name"`
	ExpiresIn *int    `json:"expires_in"`
	ExpiresAt *string `json:"expires_at"`
}

func (h *TokenHandler) Create(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[createTokenRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	in := usecase.CreateTokenInput{UserID: user.ID, Name: req.Name, ExpiresIn: req.ExpiresIn}
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		t, perr := time.Parse(time.RFC3339, strings.TrimSpace(*req.ExpiresAt))
		if perr != nil {
			return c.Error(http.StatusBadRequest, "token_expiry_invalid")
		}

		in.ExpiresAt = &t
	}

	out, err := h.tokens.Create(c.Ctx(), in)
	if err != nil {
		return mapError(c, err)
	}
	if h.access != nil {
		h.access.TokenCreate(c.R, user, req.Name)
	}

	c.W.Header().Set("Cache-Control", "no-store")

	return httpx.Created(c, out)
}

func (h *TokenHandler) List(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	items, err := h.tokens.List(c.Ctx(), user.ID)
	if err != nil {
		return mapError(c, err)
	}
	return httpx.OK(c, tokensResponse{Tokens: items})
}

func (h *TokenHandler) Revoke(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	id, err := parseID(c.Param("id"))
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_token_id")
	}

	if err := h.tokens.Revoke(c.Ctx(), user.ID, id); err != nil {
		return mapError(c, err)
	}

	if h.access != nil {
		h.access.TokenRevoke(c.R, user, strconv.FormatUint(uint64(id), 10))
	}

	return c.NoContent()
}
