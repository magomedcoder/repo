package http_test

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHTTPSmoke(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git required")
	}
	srv, dir := newTestServer(t)

	reg := doJSON(t, srv, http.MethodPost, "/api/auth/register", `{"username":"utest2","email":"utest2@example.com","password":"password1"}`, nil)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register %d %s", reg.StatusCode, reg.Body)
	}
	cookies := reg.Cookies

	created := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"hello","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create repo %d %s", created.StatusCode, created.Body)
	}

	bares, _ := filepath.Glob(filepath.Join(dir, "data", "repos", "user", "*", "hello.git"))
	if len(bares) == 0 {
		t.Fatal("bare missing")
	}

	bare := bares[0]
	seed := filepath.Join(dir, "seed")
	runGit(t, dir, "git", "clone", bare, seed)
	if err := os.WriteFile(filepath.Join(seed, "readme.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runGit(t, seed, "git", "add", ".")
	runGit(t, seed, "git", "-c", "user.email=c@d.e", "-c", "user.name=C", "commit", "-m", "init")
	runGit(t, seed, "git", "push", "origin", "HEAD:main")

	cloneURL := strings.TrimRight(srv.URL, "/") + "/utest2/hello.git"
	dest := filepath.Join(dir, "clone-http")
	cmd := exec.Command("git", "clone", cloneURL, dest)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git clone http: %v\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(dest, "readme.md")); err != nil {
		t.Fatalf("cloned file missing: %v", err)
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/utest2/hello.git/info/refs?service=git-upload-pack", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("info/refs status %d", res.StatusCode)
	}

	if res.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing security header: %v", res.Header)
	}
}
