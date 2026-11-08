package http_test

import (
	"net/http"
	"testing"
)

func TestAdminAPIRejectsNonAdmin(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "plainu", "plainu@example.com", "test-pass-1")
	srv := env.Server

	for _, path := range []string{
		"/api/admin/overview",
		"/api/admin/users",
		"/api/admin/repos",
		"/api/admin/mirrors",
		"/api/admin/access-logs",
	} {
		res := doJSON(t, srv, http.MethodGet, path, "", cookies)
		if res.StatusCode != http.StatusForbidden && res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s as non-admin: %d %s", path, res.StatusCode, res.Body)
		}
	}
}
