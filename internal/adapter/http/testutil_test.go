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

	"magomedcoder.ru/repo/internal/adapter/git"
	deliveryhttp "magomedcoder.ru/repo/internal/adapter/http"
	"magomedcoder.ru/repo/internal/adapter/http/handler"
	"magomedcoder.ru/repo/internal/adapter/sqlite"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/argon2id"
	"magomedcoder.ru/repo/pkg/token"
)

type testEnv struct {
	Server *httptest.Server
	Dir    string
	Auth   *usecase.AuthUseCase
}

func newTestServer(t *testing.T) testEnv {
	t.Helper()
	dir, authUC, router := buildTestApp(t)
	srv := httptest.NewTestServer(t, router)
	t.Cleanup(srv.Close)
	return testEnv{
		Server: srv,
		Dir:    dir,
		Auth:   authUC,
	}
}

func newLoopbackTestServer(t *testing.T) testEnv {
	t.Helper()
	dir, authUC, router := buildTestApp(t)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return testEnv{
		Server: srv,
		Dir:    dir,
		Auth:   authUC,
	}
}

func buildTestApp(t *testing.T) (dir string, authUC *usecase.AuthUseCase, router http.Handler) {
	t.Helper()
	dir = t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chdir(wd) })

	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.NewDB(dataDir)
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
	folderMemberStore := sqlite.NewFolderMemberStore(db)
	teamStore := sqlite.NewTeamStore(db)
	teamMemberStore := sqlite.NewTeamMemberStore(db)
	teamRepoGrantStore := sqlite.NewTeamRepoGrantStore(db)
	collabStore := sqlite.NewRepoCollaboratorStore(db)
	gitRepo := git.NewRepository(dataDir)
	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()

	authz := usecase.NewAuthz(folderMemberStore, teamStore, teamMemberStore, teamRepoGrantStore, collabStore)
	authUC = usecase.NewAuthUseCase(userStore, sessionStore, tokenStore, hasher, tokens)
	authUC.SetSessionSecret("0123456789abcdef0123456789abcdef")
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore, folderMemberStore, gitRepo, authz)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo, activityStore, authz)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, userStore, tokenStore, hasher, gitRepo, activityStore, authz)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore, activityStore, authz)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, gitRepo, activityStore, authz)
	membershipUC := usecase.NewAccessUseCase(folderStore, repoStore, userStore, folderMemberStore, teamStore, teamMemberStore, teamRepoGrantStore, collabStore, authz)
	profileUC := usecase.NewProfileUseCase(userStore, hasher, dataDir)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)
	adminUC := usecase.NewAdminUseCase(userStore, sessionStore, tokenStore, authUC, hasher)

	authHandler := handler.NewAuthHandler(authUC)
	authHandler.SetCSRFSecret("0123456789abcdef0123456789abcdef")
	router = deliveryhttp.NewRouter(
		authHandler,
		handler.NewRepositoryHandler(repoUC),
		handler.NewFolderHandler(folderUC),
		handler.NewMembershipHandler(membershipUC),
		handler.NewTokenHandler(tokenUC),
		handler.NewGitHandler(gitUC),
		handler.NewIssueHandler(issueUC),
		handler.NewPullRequestHandler(pullUC),
		handler.NewProfileHandler(profileUC),
		handler.NewActivityHandler(activityUC),
		handler.NewSiteHandler("Repo"),
		handler.NewAdminHandler(adminUC, nil),
		nil,
		authUC,
		false,
		"0123456789abcdef0123456789abcdef",
	)
	return dir, authUC, router
}

func loginUser(t *testing.T, env testEnv, username, email, password string) []*http.Cookie {
	t.Helper()
	if _, err := env.Auth.CreateUser(t.Context(), usecase.RegisterInput{
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
	client := srv.Client()
	if method != http.MethodGet && method != http.MethodHead {
		cookies = withCSRF(t, srv, cookies)
	}

	var reader io.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, reader)
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

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return jsonResp{
		StatusCode: res.StatusCode,
		Body:       string(raw),
		Cookies:    mergeCookies(cookies, res.Cookies()),
	}
}

func withCSRF(t *testing.T, srv *httptest.Server, cookies []*http.Cookie) []*http.Cookie {
	t.Helper()
	for _, c := range cookies {
		if c.Name == "repo_csrf" && c.Value != "" {
			return cookies
		}
	}

	client := srv.Client()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/site", nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cookies {
		req.AddCookie(c)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)

	return mergeCookies(cookies, res.Cookies())
}

func mergeCookies(existing, newer []*http.Cookie) []*http.Cookie {
	byName := make(map[string]*http.Cookie, len(existing)+len(newer))
	for _, c := range existing {
		byName[c.Name] = c
	}

	for _, c := range newer {
		byName[c.Name] = c
	}

	out := make([]*http.Cookie, 0, len(byName))
	for _, c := range byName {
		out = append(out, c)
	}

	return out
}
