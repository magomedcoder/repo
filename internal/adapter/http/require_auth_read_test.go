package http_test

import (
	"net/http"
	"testing"
)

func TestRequireAuthReadCoversRESTAndGit(t *testing.T) {
	env := newRequireAuthReadServer(t)
	cookies := loginUser(t, env, "ada", "ada@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "open", false, cookies)

	anon := []string{
		"/api/repos/" + pid,
		"/api/repos",
		"/api/search?q=op",
		"/api/search?q=op&type=code",
		"/api/activity/ada",
	}
	for _, path := range anon {
		res := doJSON(t, env.Server, http.MethodGet, path, "", nil)
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("anonymous GET %s = %d %s, want 401", path, res.StatusCode, res.Body)
		}
	}

	for _, path := range []string{"/api/repos/" + pid, "/api/repos", "/api/search?q=op", "/api/activity/ada"} {
		res := doJSON(t, env.Server, http.MethodGet, path, "", cookies)
		if res.StatusCode != http.StatusOK {
			t.Errorf("logged-in GET %s = %d %s, want 200", path, res.StatusCode, res.Body)
		}
	}

	git := doJSON(t, env.Server, http.MethodGet, "/"+pid+".git/info/refs?service=git-upload-pack", "", nil)
	if git.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous git advertise = %d, want 401", git.StatusCode)
	}
}

func TestAnonymousReadAllowedByDefault(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "ada", "ada@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "open", false, cookies)

	for _, path := range []string{"/api/repos/" + pid, "/api/repos", "/api/search?q=op", "/api/activity/ada"} {
		res := doJSON(t, env.Server, http.MethodGet, path, "", nil)
		if res.StatusCode != http.StatusOK {
			t.Errorf("anonymous GET %s = %d %s, want 200", path, res.StatusCode, res.Body)
		}
	}
}
