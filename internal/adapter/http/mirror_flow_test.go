package http_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestMirrorConfigureAndClear(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "mirroru", "mirroru@example.com", "password1")
	srv := env.Server

	created := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"app","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	cfg := doJSON(t, srv, http.MethodPut, "/api/repos/app/mirror", `{
		"enabled": true,
		"direction": "push",
		"url": "https://example.com/org/app.git",
		"username": "bot",
		"password": "secret"
	}`, cookies)
	if cfg.StatusCode != http.StatusOK {
		t.Fatalf("configure %d %s", cfg.StatusCode, cfg.Body)
	}

	if !strings.Contains(cfg.Body, `"mirror_direction":"push"`) || !strings.Contains(cfg.Body, `"mirror_has_password":true`) {
		t.Fatalf("configure body %s", cfg.Body)
	}

	cleared := doJSON(t, srv, http.MethodPut, "/api/repos/app/mirror", `{"enabled":false}`, cookies)
	if cleared.StatusCode != http.StatusOK {
		t.Fatalf("clear %d %s", cleared.StatusCode, cleared.Body)
	}

	if strings.Contains(cleared.Body, `"mirror_direction":"push"`) {
		t.Fatalf("mirror still set %s", cleared.Body)
	}
}
