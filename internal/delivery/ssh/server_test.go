package ssh_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	deliveryhttp "github.com/magomedcoder/repo/internal/delivery/http"
	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	deliveryssh "github.com/magomedcoder/repo/internal/delivery/ssh"
	"github.com/magomedcoder/repo/internal/infrastructure/git"
	"github.com/magomedcoder/repo/internal/infrastructure/persistence/sqlite"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/bcrypt"
	"github.com/magomedcoder/repo/pkg/token"
	gossh "golang.org/x/crypto/ssh"
)

func TestSSHCloneFlow(t *testing.T) {
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
	profileUC := usecase.NewProfileUseCase(userStore, filepath.Join(dir, "data", "avatars"))

	router := deliveryhttp.NewRouter(
		handler.NewAuthHandler(authUC),
		handler.NewRepositoryHandler(repoUC),
		handler.NewFolderHandler(folderUC),
		handler.NewTokenHandler(tokenUC),
		handler.NewSSHKeyHandler(sshKeyUC),
		handler.NewGitHandler(gitUC),
		handler.NewIssueHandler(issueUC),
		handler.NewPullRequestHandler(pullUC),
		handler.NewProfileHandler(profileUC),
		authUC,
	)
	httpSrv := httptest.NewServer(router)
	t.Cleanup(httpSrv.Close)

	sshSrv := deliveryssh.NewServer(sshKeyUC, gitUC, filepath.Join(dir, "data", "ssh"), "127.0.0.1:0")
	addrCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- sshSrv.ListenAndServeAnnounce(addrCh)
	}()

	var sshAddr string
	select {
	case sshAddr = <-addrCh:
	case err := <-errCh:
		t.Fatalf("ssh start: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("ssh start timeout")
	}

	reg := postJSON(t, httpSrv, "/api/auth/register", `{"username":"ada","email":"ada@example.com","password":"password1"}`, nil)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register %d %s", reg.StatusCode, reg.Body)
	}
	cookies := reg.Cookies

	created := postJSON(t, httpSrv, "/api/repos", `{"name":"hello","private":false}`, cookies)
	if created.StatusCode != http.StatusCreated && created.StatusCode != http.StatusOK {
		t.Fatalf("create repo %d %s", created.StatusCode, created.Body)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	authorized := string(gossh.MarshalAuthorizedKey(sshPub))
	payload, _ := json.Marshal(map[string]string{"title": "test", "public_key": authorized})
	keyRes := postJSON(t, httpSrv, "/api/ssh-keys", string(payload), cookies)
	if keyRes.StatusCode != http.StatusCreated {
		t.Fatalf("add key %d %s", keyRes.StatusCode, keyRes.Body)
	}

	privBlock, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}

	keyPath := filepath.Join(dir, "id_ed25519")
	pemBytes := pem.EncodeToMemory(privBlock)
	if err := os.WriteFile(keyPath, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(keyPath+".pub", []byte(authorized), 0o644); err != nil {
		t.Fatal(err)
	}

	signer, err := gossh.ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatal(err)
	}

	var createdKey struct {
		Fingerprint string `json:"fingerprint"`
	}

	if err := json.Unmarshal([]byte(keyRes.Body), &createdKey); err != nil {
		t.Fatal(err)
	}

	if got := gossh.FingerprintSHA256(signer.PublicKey()); got != createdKey.Fingerprint {
		t.Fatalf("fingerprint mismatch stored=%s file=%s auth=%s", createdKey.Fingerprint, got, gossh.FingerprintSHA256(sshPub))
	}

	sshCmd := "ssh -i " + keyPath + " -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

	seed := filepath.Join(dir, "seed")
	if err := os.MkdirAll(seed, 0o755); err != nil {
		t.Fatal(err)
	}

	run(t, seed, "git", "init", "-b", "main")
	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("ssh ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run(t, seed, "git", "add", ".")
	run(t, seed, "git", "-c", "user.email=a@b.c", "-c", "user.name=Ada", "commit", "-m", "init")
	run(t, seed, "git", "remote", "add", "origin", "ssh://git@"+sshAddr+"/ada/hello.git")
	pushOut := runOut(t, seed, "git", "-c", "core.sshCommand="+sshCmd, "push", "-u", "origin", "main")
	t.Logf("push:\n%s", pushOut)

	bares, _ := filepath.Glob(filepath.Join(dir, "data", "repos", "*", "hello.git"))
	if len(bares) == 0 {
		t.Fatal("bare repo missing on disk")
	}

	refsOut := runOut(t, dir, "git", "--git-dir="+bares[0], "show-ref")
	t.Logf("bare refs:\n%s", refsOut)
	logOut := runOut(t, dir, "git", "--git-dir="+bares[0], "log", "--oneline", "main")
	t.Logf("bare log:\n%s", logOut)
	if !bytes.Contains([]byte(logOut), []byte("init")) {
		t.Fatalf("bare repo has no commit: %s", logOut)
	}

	cloneDir := filepath.Join(dir, "clone")
	cmd := exec.Command("git", "-c", "core.sshCommand="+sshCmd, "clone", "ssh://git@"+sshAddr+"/ada/hello.git", cloneDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}

	data, err := os.ReadFile(filepath.Join(cloneDir, "README.md"))
	if err != nil || !bytes.Contains(data, []byte("ssh ok")) {
		t.Fatalf("clone content: %v %q\nclone out:\n%s", err, data, out)
	}
}

type httpResp struct {
	StatusCode int
	Body       string
	Cookies    []*http.Cookie
}

func postJSON(t *testing.T, srv *httptest.Server, path, body string, cookies []*http.Cookie) httpResp {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	return httpResp{
		StatusCode: res.StatusCode,
		Body:       string(raw),
		Cookies:    res.Cookies(),
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	_ = runOut(t, dir, name, args...)
}

func runOut(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}

	return string(out)
}
