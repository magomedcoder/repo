package http_test

import (
	"net/http"
	"testing"
)

func TestAnonymousPrivateAndMissingReposMatch(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "ada", "ada@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"secret","private":true}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	privateRepo := doJSON(t, env.Server, http.MethodGet, "/api/repos/secret", "", nil)
	missing := doJSON(t, env.Server, http.MethodGet, "/api/repos/missing", "", nil)
	if privateRepo.StatusCode != http.StatusUnauthorized || missing.StatusCode != http.StatusUnauthorized {
		t.Fatalf("private %d %s missing %d %s", privateRepo.StatusCode, privateRepo.Body, missing.StatusCode, missing.Body)
	}

	if privateRepo.Body != missing.Body {
		t.Fatalf("bodies differ private=%s missing=%s", privateRepo.Body, missing.Body)
	}
}
