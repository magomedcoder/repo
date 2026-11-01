package handler

import (
	"net/http"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/pkg/httpx"
)

type AccessHandler struct {
	access *audit.Logger
}

func NewAccessHandler(access *audit.Logger) *AccessHandler {
	return &AccessHandler{access: access}
}

type visitRequest struct {
	Path string `json:"path"`
}

func (h *AccessHandler) Visit(c *httpx.Context) error {
	if h.access == nil {
		return c.NoContent()
	}

	req, err := httpx.BindJSON[visitRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if path == "/api" || strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/assets/") {
		return c.NoContent()
	}

	user, _ := middleware.UserFromContext(c.Ctx())
	h.access.VisitPath(c.R, user, path)

	return c.NoContent()
}
