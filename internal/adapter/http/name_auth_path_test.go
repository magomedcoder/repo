package http_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestNameAuthPathMatrix(t *testing.T) {
	env := newTestServer(t)
	owner := loginUser(t, env, "ownerpath", "ownerpath@example.com", "test-pass-1")
	other := loginUser(t, env, "otherpath", "otherpath@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "taken-name", true, owner)

	t.Run("authed_create_taken", func(t *testing.T) {
		res := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"taken-name","private":false}`, other)
		if res.StatusCode != http.StatusConflict || !strings.Contains(res.Body, "repository_already_exists") {
			t.Fatalf("want 409 repository_already_exists, got %d %s", res.StatusCode, res.Body)
		}
	})

	t.Run("anon_create", func(t *testing.T) {
		res := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"new-name","private":false}`, nil)
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d %s", res.StatusCode, res.Body)
		}
	})

	t.Run("anon_get_private", func(t *testing.T) {
		res := doJSON(t, env.Server, http.MethodGet, "/api/repos/"+pid, "", nil)
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("want 404, got %d %s", res.StatusCode, res.Body)
		}
	})

	t.Run("anon_get_missing", func(t *testing.T) {
		res := doJSON(t, env.Server, http.MethodGet, "/api/repos/no-such-repo-xyz", "", nil)
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("want 404, got %d %s", res.StatusCode, res.Body)
		}
	})

	t.Run("anon_git_private", func(t *testing.T) {
		res := doJSON(t, env.Server, http.MethodGet, "/"+pid+".git/info/refs?service=git-upload-pack", "", nil)
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("want 404, got %d %s", res.StatusCode, res.Body)
		}
	})
}

func TestRequireAuthReadMissingMatchesPublic(t *testing.T) {
	env := newRequireAuthReadServer(t)
	cookies := loginUser(t, env, "ada", "ada@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "open", false, cookies)
	priv := createRepo(t, env.Server, "secret", true, cookies)

	public := doJSON(t, env.Server, http.MethodGet, "/api/repos/"+pid, "", nil)
	private := doJSON(t, env.Server, http.MethodGet, "/api/repos/"+priv, "", nil)
	missing := doJSON(t, env.Server, http.MethodGet, "/api/repos/missing-xyz", "", nil)
	for _, tc := range []struct {
		name string
		code int
	}{
		{"public", public.StatusCode},
		{"private", private.StatusCode},
		{"missing", missing.StatusCode},
	} {
		if tc.code != http.StatusUnauthorized {
			t.Fatalf("anon %s = %d, want 401", tc.name, tc.code)
		}
	}

	gitMissing := doJSON(t, env.Server, http.MethodGet, "/missing-xyz.git/info/refs?service=git-upload-pack", "", nil)
	gitPrivate := doJSON(t, env.Server, http.MethodGet, "/"+priv+".git/info/refs?service=git-upload-pack", "", nil)
	if gitMissing.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anon git missing = %d, want 401", gitMissing.StatusCode)
	}

	if gitPrivate.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anon git private = %d, want 401", gitPrivate.StatusCode)
	}
}
