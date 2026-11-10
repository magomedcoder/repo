package http_test

import (
	"context"
	json "encoding/json/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/magomedcoder/repo/internal/adapter/git"
	deliveryhttp "github.com/magomedcoder/repo/internal/adapter/http"
	"github.com/magomedcoder/repo/internal/adapter/http/handler"
	"github.com/magomedcoder/repo/internal/adapter/sqlite"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/argon2id"
	"github.com/magomedcoder/repo/pkg/secretbox"
	"github.com/magomedcoder/repo/pkg/ssrf"
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

	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.NewDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	userStore := sqlite.NewUserRepo(db)
	sessionStore := sqlite.NewSessionRepo(db)
	repoStore := sqlite.NewRepositoryRepo(db)
	tokenStore := sqlite.NewAccessTokenRepo(db)
	issueStore := sqlite.NewIssueRepo(db)
	pullStore := sqlite.NewPullRequestRepo(db)
	gitRepo := git.NewRepository(dataDir)
	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()

	activityStore := sqlite.NewActivityRepo(db)
	teamStore := sqlite.NewTeamRepo(db)
	teamMemberStore := sqlite.NewTeamMemberRepo(db)
	teamRepoGrantStore := sqlite.NewTeamRepoGrantRepo(db)
	collabStore := sqlite.NewRepoCollaboratorRepo(db)
	authz := usecase.NewAuthz(teamRepoGrantStore, collabStore)
	authUC := usecase.NewAuthUseCase(userStore, sessionStore, tokenStore, hasher, tokens)
	authUC.SetSessionSecret("0123456789abcdef0123456789abcdef")
	repoUC := usecase.NewRepositoryUseCase(repoStore, userStore, gitRepo, activityStore, authz)
	if key, err := secretbox.DeriveKey("0123456789abcdef0123456789abcdef"); err == nil {
		repoUC.SetMirrorSecretsKey(key)
	}

	repoUC.SetMirrorURLLookup(func(_ context.Context, host string) ([]net.IP, error) {
		if ssrf.HostBlockedLiteral(host) {
			return nil, ssrf.ErrBlockedHost
		}

		return []net.IP{net.ParseIP("203.0.113.10")}, nil
	})
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, userStore, tokenStore, hasher, gitRepo, activityStore, authz)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, userStore, activityStore, authz)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, userStore, gitRepo, activityStore, authz)
	membershipUC := usecase.NewAccessUseCase(repoStore, userStore, collabStore, authz)
	teamUC := usecase.NewTeamUseCase(teamStore, teamMemberStore, teamRepoGrantStore, repoStore, userStore, authz)
	profileUC := usecase.NewProfileUseCase(userStore, hasher, dataDir)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)
	adminUC := usecase.NewAdminUseCase(userStore, sessionStore, tokenStore, authUC, hasher)

	authHandler := handler.NewAuthHandler(authUC)
	authHandler.SetCSRFSecret("0123456789abcdef0123456789abcdef")
	router := deliveryhttp.NewRouter(
		authHandler,
		handler.NewRepositoryHandler(repoUC),
		handler.NewMembershipHandler(membershipUC),
		handler.NewTeamHandler(teamUC),
		handler.NewTokenHandler(tokenUC),
		handler.NewGitHandler(gitUC),
		handler.NewIssueHandler(issueUC),
		handler.NewPullRequestHandler(pullUC),
		handler.NewProfileHandler(profileUC),
		handler.NewActivityHandler(activityUC),
		handler.NewSiteHandler("Repo", ""),
		handler.NewAdminHandler(adminUC, nil),
		nil,
		authUC,
		false,
		"0123456789abcdef0123456789abcdef",
	)
	srv := httptest.NewTestServer(t, router)
	t.Cleanup(srv.Close)

	if _, err := authUC.CreateUser(t.Context(), usecase.RegisterInput{
		Username: "ada",
		Email:    "ada@example.com",
		Password: "test-pass-1",
	}); err != nil {
		t.Fatal(err)
	}
	login := doJSON(t, srv, http.MethodPost, "/api/auth/login", `{"login":"ada","password":"test-pass-1"}`, nil)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login %d %s", login.StatusCode, login.Body)
	}
	cookies := login.Cookies

	pid := createRepo(t, srv, "hello", false, cookies)

	bares, _ := filepath.Glob(filepath.Join(dir, "data", "repos", pid+".git"))
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

	pr := doJSON(t, srv, http.MethodPost, "/api/repos/"+pid+"/pulls", `{"title":"Feature","body":"adds b","base_branch":"main","head_branch":"feature"}`, cookies)
	if pr.StatusCode != http.StatusCreated {
		t.Fatalf("create pull %d %s", pr.StatusCode, pr.Body)
	}

	diff := doJSON(t, srv, http.MethodGet, "/api/repos/"+pid+"/pulls/1/diff", "", nil)
	if diff.StatusCode != http.StatusOK || !strings.Contains(diff.Body, `"can_fast_forward":true`) {
		t.Fatalf("diff %d %s", diff.StatusCode, diff.Body)
	}

	comment := doJSON(t, srv, http.MethodPost, "/api/repos/"+pid+"/pulls/1/comments", `{"body":"lgtm"}`, cookies)
	if comment.StatusCode != http.StatusCreated {
		t.Fatalf("comment %d %s", comment.StatusCode, comment.Body)
	}

	merged := doJSON(t, srv, http.MethodPost, "/api/repos/"+pid+"/pulls/1/merge", `{"strategy":"ff-only"}`, cookies)
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
