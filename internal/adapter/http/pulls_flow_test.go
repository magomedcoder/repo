package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"git.magomedcoder.ru/repo/internal/adapter/git"
	deliveryhttp "git.magomedcoder.ru/repo/internal/adapter/http"
	"git.magomedcoder.ru/repo/internal/adapter/http/handler"
	"git.magomedcoder.ru/repo/internal/adapter/sqlite"
	"git.magomedcoder.ru/repo/internal/usecase"
	"git.magomedcoder.ru/repo/pkg/argon2id"
	"git.magomedcoder.ru/repo/pkg/token"
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
	issueStore := sqlite.NewIssueStore(db)
	pullStore := sqlite.NewPullRequestStore(db)
	gitRepo := git.NewRepository()
	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()

	activityStore := sqlite.NewActivityStore(db)
	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	authUC.SetSessionSecret("0123456789abcdef0123456789abcdef")
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore, gitRepo, t.TempDir())
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo, activityStore)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, folderStore, userStore, tokenStore, hasher, gitRepo, activityStore)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore, activityStore)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, gitRepo, activityStore)
	profileUC := usecase.NewProfileUseCase(userStore, filepath.Join(dir, "data", "avatars"))
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)

	router := deliveryhttp.NewRouter(
		handler.NewAuthHandler(authUC),
		handler.NewRepositoryHandler(repoUC, filepath.Join(dir, "data", "repos")),
		handler.NewFolderHandler(folderUC),
		handler.NewTokenHandler(tokenUC),
		handler.NewGitHandler(gitUC),
		handler.NewIssueHandler(issueUC),
		handler.NewPullRequestHandler(pullUC),
		handler.NewProfileHandler(profileUC),
		handler.NewActivityHandler(activityUC),
		authUC,
		false,
	)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	if _, err := authUC.CreateUser(usecase.RegisterInput{
		Username: "ada",
		Email:    "ada@example.com",
		Password: "password1",
	}); err != nil {
		t.Fatal(err)
	}
	login := doJSON(t, srv, http.MethodPost, "/api/auth/login", `{"login":"ada","password":"password1"}`, nil)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login %d %s", login.StatusCode, login.Body)
	}
	cookies := login.Cookies

	created := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"hello","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated && created.StatusCode != http.StatusOK {
		t.Fatalf("create repo %d %s", created.StatusCode, created.Body)
	}

	bares, _ := filepath.Glob(filepath.Join(dir, "data", "repos", "user", "*", "hello.git"))
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

	pr := doJSON(t, srv, http.MethodPost, "/api/repos/hello/pulls", `{"title":"Feature","body":"adds b","base_branch":"main","head_branch":"feature"}`, cookies)
	if pr.StatusCode != http.StatusCreated {
		t.Fatalf("create pull %d %s", pr.StatusCode, pr.Body)
	}

	diff := doJSON(t, srv, http.MethodGet, "/api/repos/hello/pulls/1/diff", "", nil)
	if diff.StatusCode != http.StatusOK || !strings.Contains(diff.Body, `"can_fast_forward":true`) {
		t.Fatalf("diff %d %s", diff.StatusCode, diff.Body)
	}

	comment := doJSON(t, srv, http.MethodPost, "/api/repos/hello/pulls/1/comments", `{"body":"lgtm"}`, cookies)
	if comment.StatusCode != http.StatusCreated {
		t.Fatalf("comment %d %s", comment.StatusCode, comment.Body)
	}

	merged := doJSON(t, srv, http.MethodPost, "/api/repos/hello/pulls/1/merge", `{"strategy":"ff-only"}`, cookies)
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

func runGit(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
