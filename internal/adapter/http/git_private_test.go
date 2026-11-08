package http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func gitInfoRefs(t *testing.T, env testEnv, path, user, pass string) (int, string, string) {
	t.Helper()
	return gitInfoRefsService(t, env, path, "git-upload-pack", user, pass)
}

func gitInfoRefsService(t *testing.T, env testEnv, path, service, user, pass string) (int, string, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+path+"/info/refs?service="+service, nil)
	if err != nil {
		t.Fatal(err)
	}

	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}

	res, err := env.Server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	return res.StatusCode, res.Header.Get("WWW-Authenticate"), string(body)
}

func TestGitHTTPPrivateRepoAnonymousIsNotFound(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "gitpriv", "gitpriv@example.com", "test-pass-1")

	pid := createRepo(t, env.Server, "hidden", true, cookies)

	privStatus, privWWW, privBody := gitInfoRefs(t, env, "/"+pid+".git", "", "")
	missStatus, missWWW, missBody := gitInfoRefs(t, env, "/nosuch.git", "", "")

	if privStatus != http.StatusNotFound || privWWW != "" {
		t.Fatalf("private anonymous git status=%d www=%q body=%s", privStatus, privWWW, privBody)
	}

	if missStatus != http.StatusNotFound || missWWW != "" {
		t.Fatalf("missing anonymous git status=%d www=%q body=%s", missStatus, missWWW, missBody)
	}

	if privBody != missBody {
		t.Fatalf("anonymous bodies differ: %q vs %q", privBody, missBody)
	}
}

func TestGitHTTPBadCredentialsChallenge(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "gitbad", "gitbad@example.com", "test-pass-1")

	pid := createRepo(t, env.Server, "hidden", true, cookies)

	status, www, body := gitInfoRefs(t, env, "/"+pid+".git", "gitbad", "wrong-password")
	if status != http.StatusUnauthorized || www != `Basic realm="git"` {
		t.Fatalf("bad creds private status=%d www=%q body=%s", status, www, body)
	}

	status, www, body = gitInfoRefs(t, env, "/nosuch.git", "gitbad", "wrong-password")
	if status != http.StatusUnauthorized || www != `Basic realm="git"` {
		t.Fatalf("bad creds missing status=%d www=%q body=%s", status, www, body)
	}
}

func TestGitHTTPPrivatePushAnonymousChallenges(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "gitpush", "gitpush@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "privpush", true, cookies)

	status, www, body := gitInfoRefsService(t, env, "/"+pid+".git", "git-receive-pack", "", "")
	if status != http.StatusUnauthorized || www != `Basic realm="git"` {
		t.Fatalf("private push anon status=%d www=%q body=%s", status, www, body)
	}

	status, www, body = gitInfoRefsService(t, env, "/nosuch.git", "git-receive-pack", "", "")
	if status != http.StatusUnauthorized || www != `Basic realm="git"` {
		t.Fatalf("missing push anon status=%d www=%q body=%s", status, www, body)
	}
}

func TestGitHTTPPublicPushAnonymousChallenges(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "gitpub", "gitpub@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "pubpush", false, cookies)

	status, www, body := gitInfoRefsService(t, env, "/"+pid+".git", "git-receive-pack", "", "")
	if status != http.StatusUnauthorized || www != `Basic realm="git"` {
		t.Fatalf("public push anon status=%d www=%q body=%s", status, www, body)
	}
}

func TestGitHTTPPushOwnerPasswordOK(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "ownerpush", "ownerpush@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "owned", true, cookies)

	status, www, body := gitInfoRefsService(t, env, "/"+pid+".git", "git-receive-pack", "ownerpush", "test-pass-1")
	if status != http.StatusOK {
		t.Fatalf("owner push status=%d www=%q body=%s", status, www, body)
	}
}

func TestGitHTTPPushWriteCollabOKReadForbidden(t *testing.T) {
	env := newTestServer(t)
	ownerCookies := loginUser(t, env, "collabown", "collabown@example.com", "test-pass-1")
	_ = loginUser(t, env, "writecollab", "writecollab@example.com", "test-pass-1")
	_ = loginUser(t, env, "readcollab", "readcollab@example.com", "test-pass-1")
	_ = loginUser(t, env, "outsider", "outsider@example.com", "test-pass-1")

	pid := createRepo(t, env.Server, "shared", true, ownerCookies)

	add := doJSON(t, env.Server, http.MethodPost, "/api/repos/"+pid+"/collaborators", `{"username":"writecollab","role":"write"}`, ownerCookies)
	if add.StatusCode != http.StatusCreated {
		t.Fatalf("add write collab: %d %s", add.StatusCode, add.Body)
	}

	add = doJSON(t, env.Server, http.MethodPost, "/api/repos/"+pid+"/collaborators", `{"username":"readcollab","role":"read"}`, ownerCookies)
	if add.StatusCode != http.StatusCreated {
		t.Fatalf("add read collab: %d %s", add.StatusCode, add.Body)
	}

	status, www, body := gitInfoRefsService(t, env, "/"+pid+".git", "git-receive-pack", "writecollab", "test-pass-1")
	if status != http.StatusOK {
		t.Fatalf("write collab push status=%d www=%q body=%s", status, www, body)
	}

	status, www, body = gitInfoRefsService(t, env, "/"+pid+".git", "git-receive-pack", "readcollab", "test-pass-1")
	if status != http.StatusForbidden {
		t.Fatalf("read collab push status=%d www=%q body=%s", status, www, body)
	}

	status, www, body = gitInfoRefsService(t, env, "/"+pid+".git", "git-receive-pack", "outsider", "test-pass-1")
	if status != http.StatusForbidden {
		t.Fatalf("outsider push status=%d www=%q body=%s", status, www, body)
	}
}

func TestGitHTTPPushWithPAT(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "patuser", "patuser@example.com", "test-pass-1")
	pid := createRepo(t, env.Server, "pattarget", true, cookies)

	tok := doJSON(t, env.Server, http.MethodPost, "/api/tokens",
		fmt.Sprintf(`{"name":"push","scope":"git:write","current_password":%q}`, "test-pass-1"), cookies)
	if tok.StatusCode != http.StatusCreated {
		t.Fatalf("create token: %d %s", tok.StatusCode, tok.Body)
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(tok.Body), &out); err != nil || out.Token == "" {
		t.Fatalf("token body %s err=%v", tok.Body, err)
	}

	status, www, body := gitInfoRefsService(t, env, "/"+pid+".git", "git-receive-pack", "patuser", out.Token)
	if status != http.StatusOK {
		t.Fatalf("pat push status=%d www=%q body=%s", status, www, body)
	}
}
