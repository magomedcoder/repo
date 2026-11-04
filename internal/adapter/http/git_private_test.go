package http_test

import (
	"io"
	"net/http"
	"testing"
)

func TestGitHTTPPrivateRepoChallengesWithoutAuth(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "gitpriv", "gitpriv@example.com", "password1")

	created := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"hidden","private":true}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", created.StatusCode, created.Body)
	}

	privateRefs, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/hidden.git/info/refs?service=git-upload-pack", nil)
	if err != nil {
		t.Fatal(err)
	}

	missingRefs, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/nosuch.git/info/refs?service=git-upload-pack", nil)
	if err != nil {
		t.Fatal(err)
	}

	privRes, err := env.Server.Client().Do(privateRefs)
	if err != nil {
		t.Fatal(err)
	}
	defer privRes.Body.Close()

	privBody, _ := io.ReadAll(privRes.Body)

	missRes, err := env.Server.Client().Do(missingRefs)
	if err != nil {
		t.Fatal(err)
	}
	defer missRes.Body.Close()

	missBody, _ := io.ReadAll(missRes.Body)

	if privRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("private git status=%d www=%q body=%s", privRes.StatusCode, privRes.Header.Get("WWW-Authenticate"), privBody)
	}

	if privRes.Header.Get("WWW-Authenticate") != `Basic realm="git"` {
		t.Fatalf("private git WWW-Authenticate=%q", privRes.Header.Get("WWW-Authenticate"))
	}

	if missRes.StatusCode != http.StatusNotFound {
		t.Fatalf("missing git status=%d body=%s", missRes.StatusCode, missBody)
	}
}
