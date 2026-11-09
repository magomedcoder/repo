package usecase

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/magomedcoder/repo/internal/domain"
)

type countingHasher struct {
	memHasher
	hashes   atomic.Int32
	compares atomic.Int32
}

func (h *countingHasher) Hash(p string) (string, error) {
	h.hashes.Add(1)
	return h.memHasher.Hash(p)
}

func (h *countingHasher) Compare(hash, p string) error {
	h.compares.Add(1)
	return h.memHasher.Compare(hash, p)
}

func lockoutEnv(t *testing.T) (*AuthUseCase, *GitUseCase) {
	t.Helper()

	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "ada",
		Email:        "ada@example.com",
		PasswordHash: "h:right-password",
		Role:         domain.RoleUser,
	})
	_ = users.Create(&domain.User{
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "h:bobs-password",
		Role:         domain.RoleUser,
	})

	auth := testAuth(users, &memSessions{})
	git := NewGitUseCase(nil, users, &memAccessTokens{}, memHasher{}, nil, nil, NewAuthz(nil, nil))
	git.SetLoginGuard(auth.LoginGuard())

	return auth, git
}

func TestLoginAccountLockAcrossIPsAndEmail(t *testing.T) {
	t.Parallel()
	auth, _ := lockoutEnv(t)
	ctx := context.Background()

	for i := range loginAcctMaxFails {
		ip := fmt.Sprintf("10.0.%d.1", i)
		login := "ada"
		if i%2 == 1 {
			login = "ADA@example.com "
		}

		if _, err := auth.Login(ctx, LoginInput{
			Login:    login,
			Password: "wrong-password",
			IPKey:    ip,
			IP:       ip,
		}); !errors.Is(err, ErrInvalidCredentials) && !errors.Is(err, ErrTooManyAttempts) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}

	if _, err := auth.Login(ctx, LoginInput{
		Login:    "ada",
		Password: "right-password",
		IPKey:    "203.0.113.9",
		IP:       "203.0.113.9",
	}); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("account must be soft-locked across IPs/email: %v", err)
	}

	if _, err := auth.Login(ctx, LoginInput{
		Login:    "bob",
		Password: "bobs-password",
		IPKey:    "203.0.113.9",
		IP:       "203.0.113.9",
	}); err != nil {
		t.Fatalf("other account must stay usable: %v", err)
	}
}

func TestLoginIPWideSprayLocks(t *testing.T) {
	t.Parallel()
	auth, _ := lockoutEnv(t)
	ctx := context.Background()

	for i := range loginIPMaxFails {
		login := fmt.Sprintf("ghost%d", i)
		_, _ = auth.Login(ctx, LoginInput{
			Login:    login,
			Password: "wrong-password",
			IPKey:    "198.51.100.7",
			IP:       "198.51.100.7",
		})
	}

	if _, err := auth.Login(ctx, LoginInput{
		Login:    "bob",
		Password: "bobs-password",
		IPKey:    "198.51.100.7",
		IP:       "198.51.100.7",
	}); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("spraying many logins from one IP must lock the IP: %v", err)
	}
}

func TestLoginSuccessDoesNotResetIPCounter(t *testing.T) {
	t.Parallel()
	auth, _ := lockoutEnv(t)
	ctx := context.Background()

	for i := range loginIPMaxFails - 1 {
		_, _ = auth.Login(ctx, LoginInput{
			Login:    fmt.Sprintf("ghost%d", i),
			Password: "wrong-password",
			IPKey:    "192.0.2.1",
			IP:       "192.0.2.1",
		})
		if i%5 == 0 {
			if _, err := auth.Login(ctx, LoginInput{
				Login:    "bob",
				Password: "bobs-password",
				IPKey:    "192.0.2.1",
				IP:       "192.0.2.1",
			}); err != nil {
				t.Fatalf("own valid login: %v", err)
			}
		}
	}

	_, _ = auth.Login(ctx, LoginInput{
		Login:    "ghost-last",
		Password: "wrong-password",
		IPKey:    "192.0.2.1",
		IP:       "192.0.2.1",
	})
	if !auth.guard.IPLocked("192.0.2.1") {
		t.Fatal("interleaved valid logins must not reset the IP-wide counter")
	}
}

func TestWebAndGitShareLockoutCounters(t *testing.T) {
	t.Parallel()
	auth, git := lockoutEnv(t)
	ctx := context.Background()

	for i := range loginAcctMaxFails {
		ip := fmt.Sprintf("10.1.%d.1", i)
		if i%2 == 0 {
			_, _ = auth.Login(ctx, LoginInput{
				Login:    "ada",
				Password: "wrong-password",
				IPKey:    ip,
				IP:       ip,
			})
		} else {
			_, _ = git.Authenticate(ctx, ip, "ada", "wrong-password")
		}
	}

	if _, err := git.Authenticate(ctx, "203.0.113.50", "ada", "right-password"); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("git must see the web+git account lock: %v", err)
	}

	if _, err := auth.Login(ctx, LoginInput{
		Login:    "ada",
		Password: "right-password",
		IPKey:    "203.0.113.51",
		IP:       "203.0.113.51",
	}); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("web must see the web+git account lock: %v", err)
	}
}

func TestGitPATFailuresThrottledByIP(t *testing.T) {
	t.Parallel()
	_, git := lockoutEnv(t)
	ctx := context.Background()

	for i := range gitTokenMaxFails {
		_, err := git.Authenticate(ctx, "198.51.100.99", "", fmt.Sprintf("repo_bad%d", i))
		if !errors.Is(err, ErrUnauthorized) && !errors.Is(err, ErrTooManyAttempts) {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}

	if _, err := git.Authenticate(ctx, "198.51.100.99", "", "repo_another"); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("PAT guessing from one IP must be throttled: %v", err)
	}

	if _, err := git.Authenticate(ctx, "198.51.100.100", "", "repo_another"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("other IP must not be throttled: %v", err)
	}
}

func TestGitPasswordAuthDisabled(t *testing.T) {
	t.Parallel()
	_, git := lockoutEnv(t)
	ctx := context.Background()

	if _, err := git.Authenticate(ctx, "192.0.2.5", "ada", "right-password"); err != nil {
		t.Fatalf("password auth default on: %v", err)
	}

	git.SetPasswordAuth(false)
	if _, err := git.Authenticate(ctx, "192.0.2.5", "ada", "right-password"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("password must be refused when git_password_auth=false: %v", err)
	}
}

func TestStepUpLockoutShared(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "u",
		Email:        "u@x.io",
		PasswordHash: "h:right-password",
		Role:         domain.RoleUser,
	})

	gate := NewPasswordGate(memHasher{})
	tok := NewTokenUseCase(&memAccessTokens{}, &memTokens{})
	tok.SetStepUp(users, memHasher{})
	tok.SetPasswordGate(gate)

	profile := NewProfileUseCase(users, memHasher{}, t.TempDir())
	profile.SetPasswordGate(gate)

	ctx := context.Background()
	for i := range stepUpMaxFails - 1 {
		if _, err := tok.Create(ctx, CreateTokenInput{
			UserID:          1,
			Name:            "ci",
			CurrentPassword: "bad",
		}); !errors.Is(err, ErrCurrentPasswordWrong) {
			t.Fatalf("fail %d: %v", i, err)
		}
	}

	if _, err := profile.UpdateEmail(ctx, 1, "new@x.io", "bad", ""); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("5th shared fail must lock: %v", err)
	}

	if _, err := tok.Create(ctx, CreateTokenInput{
		UserID:          1,
		Name:            "ci",
		CurrentPassword: "right-password",
	}); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("locked user must be refused even with correct password: %v", err)
	}
}

func TestStepUpLockRevokesSessions(t *testing.T) {
	t.Parallel()
	sessions := &memSessions{}
	_ = sessions.Create(&domain.Session{
		UserID: 9,
		Token:  "tok-a",
	})
	_ = sessions.Create(&domain.Session{
		UserID: 9,
		Token:  "tok-b",
	})

	gate := NewPasswordGate(memHasher{})
	var revoked bool
	gate.SetOnLock(func(userID uint) {
		if userID != 9 {
			t.Fatalf("unexpected user %d", userID)
		}
		revoked = true
		_ = sessions.DeleteByUserID(userID)
	})

	for range stepUpMaxFails {
		_ = gate.Verify(9, "h:secret-secret", "bad", ErrCurrentPasswordWrong)
	}

	if !revoked {
		t.Fatal("onLock must run when lock engages")
	}
	if list, _ := sessions.ListByUserID(9); len(list) != 0 {
		t.Fatalf("sessions must be revoked, got %d", len(list))
	}
}

func TestStepUpSuccessResetsCounter(t *testing.T) {
	t.Parallel()
	g := NewPasswordGate(memHasher{})

	for range stepUpMaxFails - 1 {
		if err := g.Verify(7, "h:pw-pw-pw-pw", "bad", ErrCurrentPasswordWrong); !errors.Is(err, ErrCurrentPasswordWrong) {
			t.Fatal(err)
		}
	}

	if err := g.Verify(7, "h:pw-pw-pw-pw", "pw-pw-pw-pw", ErrCurrentPasswordWrong); err != nil {
		t.Fatalf("correct password: %v", err)
	}

	for range stepUpMaxFails - 1 {
		if err := g.Verify(7, "h:pw-pw-pw-pw", "bad", ErrCurrentPasswordWrong); !errors.Is(err, ErrCurrentPasswordWrong) {
			t.Fatalf("counter should have been reset: %v", err)
		}
	}
}

func TestCreateUserTakenStillHashes(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	h := &countingHasher{}
	uc := NewAuthUseCase(users, &memSessions{}, &memAccessTokens{}, h, &memTokens{})

	if _, err := uc.CreateUser(context.Background(), RegisterInput{
		Username: "ada",
		Email:    "ada@example.com",
		Password: "test-pass-1",
	}); err != nil {
		t.Fatal(err)
	}

	base := h.hashes.Load()
	if _, err := uc.CreateUser(context.Background(), RegisterInput{
		Username: "ada",
		Email:    "x@example.com",
		Password: "test-pass-1",
	}); !errors.Is(err, ErrUsernameTaken) {
		t.Fatal(err)
	}

	if _, err := uc.CreateUser(context.Background(), RegisterInput{
		Username: "eve",
		Email:    "ada@example.com",
		Password: "test-pass-1",
	}); !errors.Is(err, ErrEmailTaken) {
		t.Fatal(err)
	}

	if got := h.hashes.Load() - base; got != 2 {
		t.Fatalf("taken paths must burn one hash each, got %d", got)
	}
}

type uniqueErrUsers struct{ *memUsers }

func (uniqueErrUsers) ExistsByUsername(string) (bool, error) { return false, nil }
func (uniqueErrUsers) ExistsByEmail(string) (bool, error)    { return false, nil }
func (uniqueErrUsers) Create(*domain.User) error {
	return errors.New("UNIQUE constraint failed: users.username")
}

func TestCreateUserUniqueRaceMapsToRegistrationFailed(t *testing.T) {
	t.Parallel()
	uc := NewAuthUseCase(uniqueErrUsers{&memUsers{}}, &memSessions{}, &memAccessTokens{}, memHasher{}, &memTokens{})
	uc.SetAllowRegistration(true)

	if _, err := uc.Register(context.Background(), RegisterInput{
		Username: "ada",
		Email:    "ada@example.com",
		Password: "test-pass-1",
	}); !errors.Is(err, ErrRegistrationFailed) {
		t.Fatalf("unique violation must map to registration_failed, got %v", err)
	}
}

func TestPasswordPolicy(t *testing.T) {
	t.Parallel()

	long := make([]byte, maxPasswordLen+1)
	for i := range long {
		long[i] = 'a' + byte(i%26)
	}

	cases := []struct {
		pw   string
		want error
	}{
		{"", ErrPasswordRequired},
		{"short-pw1", ErrPasswordTooShort},
		{"ten-chars-1", nil},
		{"1234567890", ErrPasswordTooCommon},
		{"PASSWORD123", ErrPasswordTooCommon},
		{string(long), ErrPasswordTooLong},
		{string(long[:maxPasswordLen]), nil},
	}
	for _, tc := range cases {
		if err := validateNewPassword(tc.pw); !errors.Is(err, tc.want) {
			t.Errorf("validateNewPassword(%.12q) = %v, want %v", tc.pw, err, tc.want)
		}
	}
}

func TestLoginRejectsOversizedPasswordWithoutHashing(t *testing.T) {
	t.Parallel()
	h := &countingHasher{}
	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "ada",
		Email:        "a@x.io",
		PasswordHash: "h:x",
		Role:         domain.RoleUser,
	})
	uc := NewAuthUseCase(users, &memSessions{}, &memAccessTokens{}, h, &memTokens{})

	big := make([]byte, maxPasswordLen+1)
	for i := range big {
		big[i] = 'x'
	}

	if _, err := uc.Login(context.Background(), LoginInput{
		Login:    "ada",
		Password: string(big),
		IPKey:    "1.1.1.1",
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal(err)
	}

	if h.compares.Load() != 0 {
		t.Fatal("oversized password must be rejected before hashing")
	}
}

func TestRequireAuthReadAuthz(t *testing.T) {
	t.Parallel()

	public := &domain.Repository{
		ID:      1,
		OwnerID: 1,
	}
	private := &domain.Repository{
		ID:        2,
		OwnerID:   1,
		IsPrivate: true,
	}

	a := NewAuthz(nil, nil)
	if err := a.CanReadRepo(public, 0); err != nil {
		t.Fatalf("default: anonymous public read: %v", err)
	}

	a.SetRequireAuthRead(true)
	if err := a.CanReadRepo(public, 0); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("flag: anonymous public read must be 401, got %v", err)
	}

	if err := a.CanReadRepo(private, 0); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("flag: anonymous private must match public (401), got %v", err)
	}

	if err := a.CanReadRepo(public, 5); err != nil {
		t.Fatalf("flag: logged-in public read: %v", err)
	}
}

func TestRequireAuthReadSearchAndActivity(t *testing.T) {
	t.Parallel()

	authz := NewAuthz(nil, nil)
	authz.SetRequireAuthRead(true)

	uc := &RepositoryUseCase{
		authz: authz,
	}
	if _, err := uc.SearchAccessibleRepos(context.Background(), "abc", 0, 10, 0); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous repo search: %v", err)
	}

	if _, err := uc.SearchCode(context.Background(), "abc", 0, ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous code search: %v", err)
	}

	act := NewActivityUseCase(nil, nil, nil)
	act.SetAuthz(authz)
	if _, err := act.ListOwner(context.Background(), "ada", 0, 10); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous activity: %v", err)
	}
}

func TestCanOwnRepoPersonalOwnerOnly(t *testing.T) {
	t.Parallel()

	collabs := &memCollabs{}
	_ = collabs.Upsert(&domain.RepoCollaborator{
		RepoID: 1,
		UserID: 2,
		Role:   domain.RepoRoleAdmin,
	})
	a := NewAuthz(nil, collabs)
	repo := &domain.Repository{ID: 1, OwnerID: 1}

	if !a.CanOwnRepo(repo, 1) {
		t.Fatal("personal owner must own")
	}

	if a.CanOwnRepo(repo, 2) || a.CanOwnRepo(repo, 0) || a.CanOwnRepo(nil, 1) {
		t.Fatal("collaborator-admin/anonymous must not own")
	}

	if !a.CanAdminRepo(repo, 2) {
		t.Fatal("collaborator-admin keeps admin tier")
	}
}

func TestCollaboratorAdminCannotDeleteRenameOrFlipVisibility(t *testing.T) {
	t.Parallel()

	e, repo := stepUpEnv(t)
	users := e.uc.users.(*memUsers)
	users.items[1].PasswordHash = "h:devpass-1"
	collabs := &memCollabs{}
	_ = collabs.Upsert(&domain.RepoCollaborator{
		RepoID: repo.ID,
		UserID: 2,
		Role:   domain.RepoRoleAdmin,
	})
	e.uc.authz.Collabs = collabs

	ctx := context.Background()
	yes, name := true, "renamed"

	if _, err := e.uc.Update(ctx, UpdateRepositoryInput{
		Name:            "app",
		ViewerID:        2,
		Private:         &yes,
		CurrentPassword: "devpass-1",
	}); !errors.Is(err, ErrRepoOwnerOnly) {
		t.Fatalf("visibility: %v", err)
	}

	if _, err := e.uc.Update(ctx, UpdateRepositoryInput{
		Name:            "app",
		ViewerID:        2,
		NewName:         &name,
		CurrentPassword: "devpass-1",
	}); !errors.Is(err, ErrRepoOwnerOnly) {
		t.Fatalf("rename: %v", err)
	}

	desc := "collab edit"
	if _, err := e.uc.Update(ctx, UpdateRepositoryInput{
		Name:        "app",
		ViewerID:    2,
		Description: &desc,
	}); err != nil {
		t.Fatalf("collaborator-admin may still edit description: %v", err)
	}

	err := e.uc.Delete(ctx, DeleteRepositoryInput{
		Name:            "app",
		ViewerID:        2,
		Confirm:         "app",
		CurrentPassword: "devpass-1",
	})
	if !errors.Is(err, ErrRepoOwnerOnly) {
		t.Fatalf("delete: %v", err)
	}
}

func TestMirrorNeedsStepUpBehaviorChanges(t *testing.T) {
	t.Parallel()

	repo := &domain.Repository{
		MirrorDirection:      domain.MirrorPull,
		MirrorURL:            "https://git.example.com/a.git",
		MirrorForce:          true,
		MirrorPrune:          false,
		MirrorRefspecs:       "",
		MirrorPinnedHosts:    "",
		MirrorConflictPolicy: domain.MirrorConflictForce,
	}
	str := func(s string) *string { return &s }
	flag := func(b bool) *bool { return &b }

	cases := []struct {
		name string
		in   ConfigureMirrorInput
		want bool
	}{
		{"no change", ConfigureMirrorInput{
			URL:       repo.MirrorURL,
			Direction: domain.MirrorPull,
		}, false},
		{"same refspecs", ConfigureMirrorInput{
			Refspecs: str(""),
		}, false},
		{"direction to push", ConfigureMirrorInput{
			Direction: domain.MirrorPush,
		}, true},
		{"refspecs", ConfigureMirrorInput{
			Refspecs: str("+refs/heads/*:refs/heads/*"),
		}, true},
		{"pinned hosts", ConfigureMirrorInput{
			PinnedHosts: str("git.example.com=203.0.113.10"),
		}, true},
		{"conflict policy", ConfigureMirrorInput{
			ConflictPolicy: str(domain.MirrorConflictReject),
		}, true},
		{"same conflict policy", ConfigureMirrorInput{
			ConflictPolicy: str(domain.MirrorConflictForce),
		}, false},
		{"force off", ConfigureMirrorInput{
			Force: flag(false),
		}, true},
		{"force same", ConfigureMirrorInput{
			Force: flag(true),
		}, false},
		{"prune on", ConfigureMirrorInput{
			Prune: flag(true),
		}, true},
		{"insecure tls", ConfigureMirrorInput{
			InsecureTLS: flag(true),
		}, true},
	}
	for _, tc := range cases {
		if got := mirrorNeedsStepUp(repo, tc.in); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestSessionCapPrunesOldest(t *testing.T) {
	t.Parallel()
	sessions := &memSessions{}
	uc := testAuth(&memUsers{}, sessions)

	var newest string
	for range maxSessionsPerUser + 5 {
		tok, err := uc.createSession(1, "127.0.0.1", "ua")
		if err != nil {
			t.Fatal(err)
		}

		newest = tok
	}

	list, _ := sessions.ListByUserID(1)
	if len(list) != maxSessionsPerUser {
		t.Fatalf("sessions=%d want %d", len(list), maxSessionsPerUser)
	}

	if _, err := sessions.FindByToken(context.Background(), newest); err != nil {
		t.Fatalf("newest session must survive: %v", err)
	}
}

type ownerListStore struct{ *fakeStore }

func (s ownerListStore) ListByOwnerID(_ context.Context, ownerID uint, limit, _ int, _ domain.RepoOrder) ([]domain.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []domain.Repository
	for _, r := range s.repos {
		if r.OwnerID == ownerID && len(out) < limit {
			out = append(out, *r)
		}
	}

	return out, nil
}

func (s ownerListStore) ListAdmin(_ context.Context, _ domain.AdminRepoFilter) ([]domain.Repository, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []domain.Repository
	for _, r := range s.repos {
		out = append(out, *r)
	}

	return out, len(out), nil
}

type recordingRemover struct {
	removed []string
}

func (r *recordingRemover) Remove(path string) error {
	r.removed = append(r.removed, path)
	return nil
}

func adminEnv(t *testing.T) (*AdminUseCase, *memUsers, ownerListStore, *recordingRemover) {
	t.Helper()

	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "admin",
		Email:        "a@x.io",
		PasswordHash: "h:secret",
		Role:         domain.RoleAdmin,
	})
	_ = users.Create(&domain.User{
		Username:     "victim",
		Email:        "v@x.io",
		PasswordHash: "h:other",
		Role:         domain.RoleUser,
	})

	sessions := &memSessions{}
	uc := NewAdminUseCase(users, sessions, nil, NewAuthUseCase(users, sessions, nil, memHasher{}, nil), memHasher{})
	store := ownerListStore{newFakeStore()}
	rm := &recordingRemover{}
	uc.SetDataStores(store, nil)
	uc.SetRepoFiles(rm)

	return uc, users, store, rm
}

func TestAdminDeleteUserSoftDeletesOwnedRepos(t *testing.T) {
	t.Parallel()
	uc, users, store, rm := adminEnv(t)

	store.add(domain.Repository{
		Name:    "a",
		OwnerID: 2,
		Path:    "/data/a.git", IsPrivate: true,
	})
	store.add(domain.Repository{
		Name:    "b",
		OwnerID: 2,
		Path:    "/data/b.git",
	})
	keep := store.add(domain.Repository{
		Name:    "c",
		OwnerID: 1,
		Path:    "/data/c.git",
	})

	if err := uc.Delete(context.Background(), 1, 2, "secret"); err != nil {
		t.Fatal(err)
	}

	if len(store.repos) != 1 || store.repos[keep.ID] == nil {
		t.Fatalf("only the deleted user's repos must go: %v", store.repos)
	}

	if len(rm.removed) != 2 {
		t.Fatalf("on-disk repos must be removed: %v", rm.removed)
	}

	if _, err := users.FindByID(context.Background(), 2); err == nil {
		t.Fatal("user must be deleted")
	}
}

func TestAdminListReposHidesPrivateDescription(t *testing.T) {
	t.Parallel()
	uc, _, store, _ := adminEnv(t)

	store.add(domain.Repository{
		Name:        "pub",
		OwnerID:     2,
		Description: "public words",
	})
	store.add(domain.Repository{
		Name:        "priv",
		OwnerID:     2,
		IsPrivate:   true,
		Description: "secret words",
	})

	res, err := uc.ListRepos(context.Background(), 1, domain.AdminRepoFilter{})
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range res.Repos {
		switch item.Name {
		case "pub":
			if item.Description != "public words" {
				t.Fatalf("public description: %q", item.Description)
			}
		case "priv":
			if item.Description != "" {
				t.Fatalf("private description leaked to admin list: %q", item.Description)
			}
		}
	}
}

func TestAdminPasswordResetPolicyAndStepUp(t *testing.T) {
	t.Parallel()
	uc, users, _, _ := adminEnv(t)

	weak := "short"
	if _, err := uc.Update(context.Background(), 1, 2, AdminUpdateInput{Password: &weak, CurrentPassword: "secret"}); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("policy: %v", err)
	}

	next := "a-much-better-pass"
	if _, err := uc.Update(context.Background(), 1, 2, AdminUpdateInput{Password: &next, CurrentPassword: "secret"}); err != nil {
		t.Fatal(err)
	}

	u, _ := users.FindByID(context.Background(), 2)
	if u.PasswordHash != "h:"+next {
		t.Fatalf("hash not updated: %q", u.PasswordHash)
	}
}
