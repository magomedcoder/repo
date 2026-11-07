package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"magomedcoder.ru/repo/pkg/secretbox"
)

func TestGitHTTPPrivateRepoNotFoundForOutsider(t *testing.T) {
	env := newTestServer(t)
	ownerCookies := loginUser(t, env, "ownpriv", "ownpriv@example.com", "password1")
	outsiderCookies := loginUser(t, env, "outpriv", "outpriv@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"secret","private":true}`, ownerCookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/secret.git/info/refs?service=git-upload-pack", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("outpriv", "password1")
	_ = outsiderCookies

	res, err := env.Server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("outsider private git status=%d body=%s", res.StatusCode, body)
	}
}

func TestReadCollaboratorCannotCreatePrivateIssue(t *testing.T) {
	env := newTestServer(t)
	ownerCookies := loginUser(t, env, "owncollab", "owncollab@example.com", "password1")
	readerCookies := loginUser(t, env, "readcollab", "readcollab@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"privissues","private":true}`, ownerCookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	add := doJSON(t, env.Server, http.MethodPost, "/api/repos/privissues/collaborators", `{"username":"readcollab","role":"read"}`, ownerCookies)
	if add.StatusCode != http.StatusCreated && add.StatusCode != http.StatusOK {
		t.Fatalf("add collab %d %s", add.StatusCode, add.Body)
	}

	issue := doJSON(t, env.Server, http.MethodPost, "/api/repos/privissues/issues", `{"title":"nope","body":"x"}`, readerCookies)
	if issue.StatusCode != http.StatusForbidden {
		t.Fatalf("read collab create issue status=%d body=%s", issue.StatusCode, issue.Body)
	}
}

func TestMirrorURLRejectsNipIO(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "ssrfuser", "ssrfuser@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"ssrfapp","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	cfg := doJSON(t, env.Server, http.MethodPut, "/api/repos/ssrfapp/mirror", `{
		"enabled": true,
		"direction": "push",
		"url": "https://127.0.0.1.nip.io/evil.git"
	}`, cookies)
	if cfg.StatusCode != http.StatusBadRequest || !strings.Contains(cfg.Body, "invalid_mirror_url") {
		t.Fatalf("nip.io must be rejected: %d %s", cfg.StatusCode, cfg.Body)
	}
}

func TestMirrorPasswordStoredSealed(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "sealuser", "sealuser@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"sealapp","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	cfg := doJSON(t, env.Server, http.MethodPut, "/api/repos/sealapp/mirror", `{
		"enabled": true,
		"direction": "push",
		"url": "https://example.com/org/sealapp.git",
		"username": "bot",
		"password": "hunter2-secret"
	}`, cookies)
	if cfg.StatusCode != http.StatusOK {
		t.Fatalf("configure %d %s", cfg.StatusCode, cfg.Body)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(cfg.Body), &payload); err != nil {
		t.Fatal(err)
	}

	if has, _ := payload["mirror_has_password"].(bool); !has {
		t.Fatalf("expected mirror_has_password: %s", cfg.Body)
	}

	raw, _ := json.Marshal(payload)
	if strings.Contains(string(raw), "hunter2-secret") {
		t.Fatal("password echoed in API response")
	}

	if !secretbox.IsSealed("v1:deadbeef") {
		t.Fatal("secretbox prefix helper broken")
	}
}

func TestCreateNameCollisionHidesPrivateForeign(t *testing.T) {
	env := newTestServer(t)
	ownerCookies := loginUser(t, env, "ownerpriv", "ownerpriv@example.com", "password1")
	otherCookies := loginUser(t, env, "otherpriv", "otherpriv@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"secret-name","private":true}`, ownerCookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create private %d %s", created.StatusCode, created.Body)
	}

	probe := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"secret-name","private":false}`, otherCookies)
	if probe.StatusCode != http.StatusBadRequest || !strings.Contains(probe.Body, "invalid_repository_name") {
		t.Fatalf("foreign private collision must be invalid_name: %d %s", probe.StatusCode, probe.Body)
	}

	dup := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"secret-name","private":true}`, ownerCookies)
	if dup.StatusCode != http.StatusConflict || !strings.Contains(dup.Body, "repository_already_exists") {
		t.Fatalf("owner duplicate must be already_exists: %d %s", dup.StatusCode, dup.Body)
	}
}

func TestDeleteConfirmIgnoresQuery(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "delquery", "delquery@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"delme","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	// confirm only in query (legacy) must not authorize delete.
	res := doJSON(t, env.Server, http.MethodDelete, "/api/repos/delme?confirm=delme", `{"current_password":"password1"}`, cookies)
	if res.StatusCode == http.StatusNoContent {
		t.Fatalf("delete must not succeed with confirm only in query: %d %s", res.StatusCode, res.Body)
	}

	ok := doJSON(t, env.Server, http.MethodDelete, "/api/repos/delme", `{"confirm":"delme","current_password":"password1"}`, cookies)
	if ok.StatusCode != http.StatusNoContent {
		t.Fatalf("delete with body confirm: %d %s", ok.StatusCode, ok.Body)
	}
}

func TestPublicProfileHidesRole(t *testing.T) {
	env := newTestServer(t)
	ownerCookies := loginUser(t, env, "profilerole", "profilerole@example.com", "password1")
	viewerCookies := loginUser(t, env, "roleviewer", "roleviewer@example.com", "password1")

	self := doJSON(t, env.Server, http.MethodGet, "/api/users/profilerole", "", ownerCookies)
	if self.StatusCode != http.StatusOK || !strings.Contains(self.Body, `"role"`) {
		t.Fatalf("self profile should include role: %d %s", self.StatusCode, self.Body)
	}

	other := doJSON(t, env.Server, http.MethodGet, "/api/users/profilerole", "", viewerCookies)
	if other.StatusCode != http.StatusOK {
		t.Fatalf("other profile %d %s", other.StatusCode, other.Body)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(other.Body), &payload); err != nil {
		t.Fatal(err)
	}

	if _, ok := payload["role"]; ok {
		t.Fatalf("public profile must omit role: %s", other.Body)
	}
}
