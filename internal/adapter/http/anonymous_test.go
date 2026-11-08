package http_test

import (
	"net/http"
	"testing"
)

func TestAnonymousPrivateAndMissingReposMatch(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "ada", "ada@example.com", "test-pass-1")

	pid := createRepo(t, env.Server, "secret", true, cookies)

	privateRepo := doJSON(t, env.Server, http.MethodGet, "/api/repos/"+pid, "", nil)
	missing := doJSON(t, env.Server, http.MethodGet, "/api/repos/missing", "", nil)
	if privateRepo.StatusCode != http.StatusNotFound || missing.StatusCode != http.StatusNotFound {
		t.Fatalf("private %d %s missing %d %s", privateRepo.StatusCode, privateRepo.Body, missing.StatusCode, missing.Body)
	}

	if privateRepo.Body != missing.Body {
		t.Fatalf("bodies differ private=%s missing=%s", privateRepo.Body, missing.Body)
	}
}
