package http_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"repo.magomedcoder.ru/repo/internal/adapter/git"
	deliveryhttp "repo.magomedcoder.ru/repo/internal/adapter/http"
	"repo.magomedcoder.ru/repo/internal/adapter/http/handler"
	"repo.magomedcoder.ru/repo/internal/adapter/sqlite"
	"repo.magomedcoder.ru/repo/internal/usecase"
	"repo.magomedcoder.ru/repo/pkg/argon2id"
	"repo.magomedcoder.ru/repo/pkg/token"
)

type testEnv struct {
	Server *httptest.Server
	Dir    string
	Auth   *usecase.AuthUseCase
}

func newTestServer(t *testing.T) testEnv {
	t.Helper()
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
	activityStore := sqlite.NewActivityStore(db)
	gitRepo := git.NewRepository()
	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()
	avatarsDir := filepath.Join(dir, "data", "avatars")
	reposRoot := filepath.Join(dir, "data", "repos")

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	authUC.SetSessionSecret("0123456789abcdef0123456789abcdef")
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore, gitRepo, t.TempDir())
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo, activityStore)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, userStore, tokenStore, hasher, gitRepo, activityStore)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore, activityStore)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, gitRepo, activityStore)
	profileUC := usecase.NewProfileUseCase(userStore, avatarsDir)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)

	router := deliveryhttp.NewRouter(
		handler.NewAuthHandler(authUC),
		handler.NewRepositoryHandler(repoUC, reposRoot),
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
	return testEnv{Server: srv, Dir: dir, Auth: authUC}
}

func loginUser(t *testing.T, env testEnv, username, email, password string) []*http.Cookie {
	t.Helper()
	if _, err := env.Auth.CreateUser(usecase.RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	}); err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf(`{"login":%q,"password":%q}`, username, password)
	res := doJSON(t, env.Server, http.MethodPost, "/api/auth/login", body, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login %d %s", res.StatusCode, res.Body)
	}
	return res.Cookies
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
		if c.Name == "repo_csrf" && c.Value != "" {
			req.Header.Set("X-CSRF-Token", c.Value)
		}
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
