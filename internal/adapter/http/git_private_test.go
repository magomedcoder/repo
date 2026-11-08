package http_test

import (
	"io"
	"net/http"
	"testing"
)

func gitInfoRefs(t *testing.T, env testEnv, path, user, pass string) (int, string, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+path+"/info/refs?service=git-upload-pack", nil)
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
	cookies := loginUser(t, env, "gitpriv", "gitpriv@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"hidden","private":true}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	privStatus, privWWW, privBody := gitInfoRefs(t, env, "/hidden.git", "", "")
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
	cookies := loginUser(t, env, "gitbad", "gitbad@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"hidden","private":true}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	status, www, body := gitInfoRefs(t, env, "/hidden.git", "gitbad", "wrong-password")
	if status != http.StatusUnauthorized || www != `Basic realm="git"` {
		t.Fatalf("bad creds private status=%d www=%q body=%s", status, www, body)
	}

	status, www, body = gitInfoRefs(t, env, "/nosuch.git", "gitbad", "wrong-password")
	if status != http.StatusUnauthorized || www != `Basic realm="git"` {
		t.Fatalf("bad creds missing status=%d www=%q body=%s", status, www, body)
	}
}
