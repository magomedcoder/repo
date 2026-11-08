package http_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"magomedcoder.ru/repo/internal/usecase"
)

func TestPrivateBrowseEndpointsHiddenWithoutAccess(t *testing.T) {
	env := newTestServer(t)
	owner := loginUser(t, env, "idorowner", "idorowner@example.com", "password1")
	outsider := loginUser(t, env, "idorother", "idorother@example.com", "password1")

	if r := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"vault","private":true}`, owner); r.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", r.StatusCode, r.Body)
	}

	actions := []string{
		"blob?path=README.md",
		"raw?path=README.md",
		"blame?path=README.md",
		"tree",
		"commits",
		"branches",
		"tags",
		"summary",
	}

	for _, viewer := range []struct {
		name    string
		cookies []*http.Cookie
	}{
		{"anonymous", nil},
		{"outsider", outsider},
	} {
		for _, action := range actions {
			t.Run(viewer.name+"/"+action, func(t *testing.T) {
				got := doJSON(t, env.Server, http.MethodGet, "/api/repos/vault/"+action, "", viewer.cookies)
				missing := doJSON(t, env.Server, http.MethodGet, "/api/repos/nosuchrepo/"+action, "", viewer.cookies)

				if got.StatusCode != http.StatusNotFound {
					t.Fatalf("status=%d body=%s want 404", got.StatusCode, got.Body)
				}

				if got.StatusCode != missing.StatusCode || got.Body != missing.Body {
					t.Fatalf("private differs from missing: %d %q vs %d %q", got.StatusCode, got.Body, missing.StatusCode, missing.Body)
				}
			})
		}
	}
}

func TestRevokeForeignTokenNotFound(t *testing.T) {
	env := newTestServer(t)
	alice := loginUser(t, env, "tokalice", "tokalice@example.com", "password1")
	bob := loginUser(t, env, "tokbob", "tokbob@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/tokens", `{"name":"ci","current_password":"password1"}`, alice)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create token %d %s", created.StatusCode, created.Body)
	}

	idx := strings.Index(created.Body, `"id":`)
	if idx < 0 {
		t.Fatalf("no id in %s", created.Body)
	}

	rest := created.Body[idx+len(`"id":`):]
	end := strings.IndexAny(rest, ",}")
	id := rest[:end]

	res := doJSON(t, env.Server, http.MethodDelete, "/api/tokens/"+id, `{"current_password":"password1"}`, bob)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("bob revoke alice token: %d %s", res.StatusCode, res.Body)
	}

	list := doJSON(t, env.Server, http.MethodGet, "/api/tokens", "", alice)
	if !strings.Contains(list.Body, `"name":"ci"`) {
		t.Fatalf("alice token must survive: %s", list.Body)
	}
}

func TestAuthMeSessionStates(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "meuser", "meuser@example.com", "password1")

	do := func(cookie *http.Cookie) (*http.Response, string) {
		t.Helper()

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/api/auth/me", nil)
		if err != nil {
			t.Fatal(err)
		}

		if cookie != nil {
			req.AddCookie(cookie)
		}

		res, err := env.Server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		body, _ := io.ReadAll(res.Body)

		return res, strings.TrimSpace(string(body))
	}

	if res, body := do(nil); res.StatusCode != http.StatusOK || body != "null" {
		t.Fatalf("anonymous me: %d %q", res.StatusCode, body)
	}

	res, body := do(&http.Cookie{Name: "session", Value: "bogus-token"})
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad cookie me: %d %q", res.StatusCode, body)
	}

	cleared := false
	for _, c := range res.Cookies() {
		if c.Name == "session" && c.MaxAge < 0 && c.Value == "" {
			cleared = true
		}
	}

	if !cleared {
		t.Fatalf("session cookie not cleared: %v", res.Header.Values("Set-Cookie"))
	}

	for _, c := range cookies {
		if c.Name != "session" {
			continue
		}

		if res, body := do(c); res.StatusCode != http.StatusOK || !strings.Contains(body, "meuser") {
			t.Fatalf("valid me: %d %q", res.StatusCode, body)
		}
	}
}

func TestRevokeAllSessionsRequiresPassword(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "revall", "revall@example.com", "password1")

	for _, tc := range []struct {
		name string
		body string
		want int
	}{
		{"missing password", ``, http.StatusBadRequest},
		{"wrong password", `{"current_password":"nope-nope"}`, http.StatusBadRequest},
		{"correct password", `{"current_password":"password1"}`, http.StatusNoContent},
	} {
		res := doJSON(t, env.Server, http.MethodPost, "/api/auth/sessions/revoke-all", tc.body, cookies)
		if res.StatusCode != tc.want {
			t.Fatalf("%s: %d %s want %d", tc.name, res.StatusCode, res.Body, tc.want)
		}
	}
}

func TestTeamGrantRequiresRepoAdmin(t *testing.T) {
	env := newTestServer(t)
	owner := loginUser(t, env, "tgowner", "tgowner@example.com", "password1")
	mallory := loginUser(t, env, "tgmallory", "tgmallory@example.com", "password1")

	if r := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"tgvault","private":true}`, owner); r.StatusCode != http.StatusCreated {
		t.Fatalf("create repo %d %s", r.StatusCode, r.Body)
	}

	if r := doJSON(t, env.Server, http.MethodPost, "/api/teams", `{"name":"Mallory Crew"}`, mallory); r.StatusCode != http.StatusCreated {
		t.Fatalf("create team %d %s", r.StatusCode, r.Body)
	}

	res := doJSON(t, env.Server, http.MethodPost, "/api/teams/mallory-crew/repos", `{"repo":"tgvault","role":"admin"}`, mallory)
	if res.StatusCode != http.StatusNotFound && res.StatusCode != http.StatusForbidden {
		t.Fatalf("outsider team grant: %d %s", res.StatusCode, res.Body)
	}

	if r := doJSON(t, env.Server, http.MethodGet, "/api/repos/tgvault", "", mallory); r.StatusCode != http.StatusNotFound {
		t.Fatalf("mallory must still not see repo: %d %s", r.StatusCode, r.Body)
	}

	if r := doJSON(t, env.Server, http.MethodPost, "/api/teams", `{"name":"Writers"}`, owner); r.StatusCode != http.StatusCreated {
		t.Fatalf("create writers team %d %s", r.StatusCode, r.Body)
	}

	if r := doJSON(t, env.Server, http.MethodPost, "/api/teams/writers/members", `{"username":"tgmallory"}`, owner); r.StatusCode != http.StatusCreated {
		t.Fatalf("add member %d %s", r.StatusCode, r.Body)
	}

	if r := doJSON(t, env.Server, http.MethodPost, "/api/teams/writers/repos", `{"repo":"tgvault","role":"write"}`, owner); r.StatusCode != http.StatusOK {
		t.Fatalf("owner grant %d %s", r.StatusCode, r.Body)
	}

	res = doJSON(t, env.Server, http.MethodPost, "/api/teams/mallory-crew/repos", `{"repo":"tgvault","role":"admin"}`, mallory)
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
		t.Fatalf("write collaborator granted team on repo: %d %s", res.StatusCode, res.Body)
	}
}

func TestLoginLogoutRotateCSRF(t *testing.T) {
	env := newTestServer(t)

	if _, err := env.Auth.CreateUser(t.Context(), usecase.RegisterInput{
		Username: "csrfuser",
		Email:    "csrfuser@example.com",
		Password: "password1",
	}); err != nil {
		t.Fatal(err)
	}

	anon := doJSON(t, env.Server, http.MethodGet, "/api/auth/csrf", "", nil)
	before := cookieValue(anon.Cookies, "repo_csrf")
	if before == "" {
		t.Fatalf("no anonymous csrf cookie: %v", anon.Cookies)
	}

	login := doJSON(t, env.Server, http.MethodPost, "/api/auth/login", `{"login":"csrfuser","password":"password1"}`, anon.Cookies)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login %d %s", login.StatusCode, login.Body)
	}

	afterLogin := cookieValue(login.Cookies, "repo_csrf")
	if afterLogin == "" || afterLogin == before {
		t.Fatalf("login must rotate csrf: before=%q after=%q", before, afterLogin)
	}

	logout := doJSON(t, env.Server, http.MethodPost, "/api/auth/logout", "", login.Cookies)
	if logout.StatusCode != http.StatusOK && logout.StatusCode != http.StatusNoContent {
		t.Fatalf("logout %d %s", logout.StatusCode, logout.Body)
	}

	afterLogout := cookieValue(logout.Cookies, "repo_csrf")
	if afterLogout == "" || afterLogout == afterLogin {
		t.Fatalf("logout must rotate csrf: login=%q logout=%q", afterLogin, afterLogout)
	}
}

func cookieValue(cookies []*http.Cookie, name string) string {
	for _, c := range cookies {
		if c.Name == name {
			return c.Value
		}
	}

	return ""
}
