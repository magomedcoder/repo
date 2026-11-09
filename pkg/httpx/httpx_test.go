package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/magomedcoder/repo/pkg/httpx"
)

func TestRouterOK(t *testing.T) {
	t.Parallel()
	r := httpx.New()
	r.GET("/ping", func(c *httpx.Context) error {
		return httpx.OK(c, map[string]string{"ok": "1"})
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestGroupMiddleware(t *testing.T) {
	t.Parallel()
	var saw bool
	r := httpx.New()
	g := r.Group("/api", func(next httpx.Handler) httpx.Handler {
		return func(c *httpx.Context) error {
			saw = true
			return next(c)
		}
	})
	g.GET("/x", func(c *httpx.Context) error {
		return c.NoContent()
	})

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)
	if !saw || rec.Code != http.StatusNoContent {
		t.Fatalf("saw=%v status=%d", saw, rec.Code)
	}
}
