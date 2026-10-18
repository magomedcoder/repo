package http_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"git.magomedcoder.ru/repo/internal/adapter/git"
	deliveryhttp "git.magomedcoder.ru/repo/internal/adapter/http"
	"git.magomedcoder.ru/repo/internal/adapter/http/handler"
	"git.magomedcoder.ru/repo/internal/adapter/sqlite"
	"git.magomedcoder.ru/repo/internal/usecase"
	"git.magomedcoder.ru/repo/pkg/bcrypt"
	"git.magomedcoder.ru/repo/pkg/token"
)

func newTestServer(t *testing.T) (*httptest.Server, string) {
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
	hasher := bcrypt.NewHasher()
	tokens := token.NewGenerator()
	avatarsDir := filepath.Join(dir, "data", "avatars")
	reposRoot := filepath.Join(dir, "data", "repos")

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo, activityStore)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, folderStore, userStore, tokenStore, hasher, gitRepo, activityStore)
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
	)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, dir
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
