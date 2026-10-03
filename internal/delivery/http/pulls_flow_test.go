package http_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	deliveryhttp "github.com/magomedcoder/repo/internal/delivery/http"
	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	"github.com/magomedcoder/repo/internal/infrastructure/git"
	"github.com/magomedcoder/repo/internal/infrastructure/persistence/sqlite"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/bcrypt"
	"github.com/magomedcoder/repo/pkg/token"
)

func TestPullRequestFlow(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git required")
	}

	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.NewDB(filepath.Join(dir, "data", "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}

	userStore := sqlite.NewUserStore(db)
	sessionStore := sqlite.NewSessionStore(db)
	folderStore := sqlite.NewFolderStore(db)
	repoStore := sqlite.NewRepositoryStore(db)
	tokenStore := sqlite.NewAccessTokenStore(db)
	sshKeyStore := sqlite.NewSSHKeyStore(db)
	issueStore := sqlite.NewIssueStore(db)
	pullStore := sqlite.NewPullRequestStore(db)
	gitRepo := git.NewRepository()
	hasher := bcrypt.NewHasher()
	tokens := token.NewGenerator()

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	sshKeyUC := usecase.NewSSHKeyUseCase(sshKeyStore, userStore)
	gitUC := usecase.NewGitUseCase(repoStore, folderStore, userStore, tokenStore, hasher, gitRepo)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, gitRepo)

	router := deliveryhttp.NewRouter(
		handler.NewAuthHandler(authUC),
		handler.NewRepositoryHandler(repoUC),
		handler.NewFolderHandler(folderUC),
		handler.NewTokenHandler(tokenUC),
		handler.NewSSHKeyHandler(sshKeyUC),
		handler.NewGitHandler(gitUC),
		handler.NewIssueHandler(issueUC),
		handler.NewPullRequestHandler(pullUC),
		authUC,
	)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	reg := doJSON(t, srv, http.MethodPost, "/api/auth/register", `{"username":"ada","email":"ada@example.com","password":"password1"}`, nil)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register %d %s", reg.StatusCode, reg.Body)
	}
	cookies := reg.Cookies

	created := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"hello","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated && created.StatusCode != http.StatusOK {
		t.Fatalf("create repo %d %s", created.StatusCode, created.Body)
	}

	bares, _ := filepath.Glob(filepath.Join(dir, "data", "repos", "*", "hello.git"))
	if len(bares) == 0 {
		t.Fatal("bare missing")
	}

	bare := bares[0]
	seed := filepath.Join(dir, "seed")
	runGit(t, dir, "git", "clone", bare, seed)
	if err := os.WriteFile(filepath.Join(seed, "a.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runGit(t, seed, "git", "add", ".")
	runGit(t, seed, "git", "-c", "user.email=a@b.c", "-c", "user.name=A", "commit", "-m", "base")
	runGit(t, seed, "git", "push", "origin", "main")
	runGit(t, seed, "git", "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(seed, "b.txt"), []byte("feat\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runGit(t, seed, "git", "add", ".")
	runGit(t, seed, "git", "-c", "user.email=a@b.c", "-c", "user.name=A", "commit", "-m", "feat")
	runGit(t, seed, "git", "push", "-u", "origin", "feature")

	pr := doJSON(t, srv, http.MethodPost, "/api/repos/ada/hello/pulls", `{"title":"Feature","body":"adds b","base_branch":"main","head_branch":"feature"}`, cookies)
	if pr.StatusCode != http.StatusCreated {
		t.Fatalf("create pull %d %s", pr.StatusCode, pr.Body)
	}

	diff := doJSON(t, srv, http.MethodGet, "/api/repos/ada/hello/pulls/1/diff", "", nil)
	if diff.StatusCode != http.StatusOK || !strings.Contains(diff.Body, `"can_fast_forward":true`) {
		t.Fatalf("diff %d %s", diff.StatusCode, diff.Body)
	}

	comment := doJSON(t, srv, http.MethodPost, "/api/repos/ada/hello/pulls/1/comments", `{"body":"lgtm"}`, cookies)
	if comment.StatusCode != http.StatusCreated {
		t.Fatalf("comment %d %s", comment.StatusCode, comment.Body)
	}

	merged := doJSON(t, srv, http.MethodPost, "/api/repos/ada/hello/pulls/1/merge", `{"strategy":"ff-only"}`, cookies)
	if merged.StatusCode != http.StatusOK {
		t.Fatalf("merge %d %s", merged.StatusCode, merged.Body)
	}

	var item struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(merged.Body), &item); err != nil || item.State != "merged" {
		t.Fatalf("merged body %s", merged.Body)
	}
}

type jsonResp struct {
	StatusCode int
	Body       string
	Cookies    []*http.Cookie
}

func doJSON(t *testing.T, srv *httptest.Server, method, path, body string, cookies []*http.Cookie) jsonResp {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequest(method, srv.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, c := range cookies {
		req.AddCookie(c)
	}

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return jsonResp{
		StatusCode: res.StatusCode,
		Body:       string(raw),
		Cookies:    res.Cookies(),
	}
}

func runGit(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
