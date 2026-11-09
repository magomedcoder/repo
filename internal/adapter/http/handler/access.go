package handler

import (
	"net/http"
	"strings"

	"github.com/magomedcoder/repo/internal/adapter/http/audit"
	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/pkg/httpx"
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
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}

	if path == "" {
		path = "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if len(path) > 512 || strings.Contains(path, "..") || strings.Contains(path, "//") ||
		strings.ContainsAny(path, "\\%") || strings.IndexFunc(path, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
		return c.NoContent()
	}

	if path == "/api" || strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/assets/") {
		return c.NoContent()
	}

	user, _ := middleware.UserFromContext(c.Ctx())
	h.access.VisitClientPath(c.R, user, path)

	return c.NoContent()
}
