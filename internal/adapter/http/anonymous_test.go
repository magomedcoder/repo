package http_test

import (
	"net/http"
	"testing"
)

func TestAnonymousPrivateAndMissingReposMatch(t *testing.T) {
	srv, _ := newTestServer(t)
	reg := doJSON(t, srv, http.MethodPost, "/api/auth/register", `{"username":"ada","email":"ada@example.com","password":"password1"}`, nil)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register %d %s", reg.StatusCode, reg.Body)
	}

	created := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"secret","private":true}`, reg.Cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	privateRepo := doJSON(t, srv, http.MethodGet, "/api/repos/ada/secret", "", nil)
	missing := doJSON(t, srv, http.MethodGet, "/api/repos/ada/missing", "", nil)
	if privateRepo.StatusCode != http.StatusUnauthorized || missing.StatusCode != http.StatusUnauthorized {
		t.Fatalf("private %d %s missing %d %s", privateRepo.StatusCode, privateRepo.Body, missing.StatusCode, missing.Body)
	}

	if privateRepo.Body != missing.Body {
		t.Fatalf("bodies differ private=%s missing=%s", privateRepo.Body, missing.Body)
	}
}
