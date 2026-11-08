package http_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestMirrorConfigureAndClear(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "mirroru", "mirroru@example.com", "test-pass-1")
	srv := env.Server

	pid := createRepo(t, srv, "app", false, cookies)

	cfg := doJSON(t, srv, http.MethodPut, "/api/repos/"+pid+"/mirror", `{
		"current_password": "test-pass-1",
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

	paused := doJSON(t, srv, http.MethodPut, "/api/repos/"+pid+"/mirror", `{"enabled":false,"direction":"push","url":"https://example.com/org/app.git"}`, cookies)
	if paused.StatusCode != http.StatusOK {
		t.Fatalf("pause %d %s", paused.StatusCode, paused.Body)
	}

	if !strings.Contains(paused.Body, `"mirror_status":"paused"`) || !strings.Contains(paused.Body, `"mirror_has_password":true`) ||
		!strings.Contains(paused.Body, `"mirror_direction":"push"`) {
		t.Fatalf("pause must keep config: %s", paused.Body)
	}

	cleared := doJSON(t, srv, http.MethodPut, "/api/repos/"+pid+"/mirror", `{"clear":true}`, cookies)
	if cleared.StatusCode != http.StatusOK {
		t.Fatalf("clear %d %s", cleared.StatusCode, cleared.Body)
	}

	if strings.Contains(cleared.Body, `"mirror_direction":"push"`) {
		t.Fatalf("mirror still set %s", cleared.Body)
	}
}

func TestMirrorStatusHistoryAndAdminGuard(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "mirrorv", "mirrorv@example.com", "test-pass-1")
	srv := env.Server

	svc := createRepo(t, srv, "svc", false, cookies)

	if r := doJSON(t, srv, http.MethodPut, "/api/repos/"+svc+"/mirror", `{"current_password":"test-pass-1","enabled":true,"direction":"push","url":"https://example.com/o/svc.git","interval":"1h","conflict_policy":"reject"}`, cookies); r.StatusCode != http.StatusOK {
		t.Fatalf("configure %d %s", r.StatusCode, r.Body)
	}

	st := doJSON(t, srv, http.MethodGet, "/api/repos/"+svc+"/mirror", "", cookies)
	if st.StatusCode != http.StatusOK || !strings.Contains(st.Body, `"direction":"push"`) {
		t.Fatalf("status %d %s", st.StatusCode, st.Body)
	}

	hist := doJSON(t, srv, http.MethodGet, "/api/repos/"+svc+"/mirror/history", "", cookies)
	if hist.StatusCode != http.StatusOK {
		t.Fatalf("history %d %s", hist.StatusCode, hist.Body)
	}

	adm := doJSON(t, srv, http.MethodGet, "/api/admin/mirrors", "", cookies)
	if adm.StatusCode != http.StatusForbidden && adm.StatusCode != http.StatusUnauthorized {
		t.Fatalf("admin mirrors %d %s", adm.StatusCode, adm.Body)
	}
}
