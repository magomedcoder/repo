package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

func TestNormalizeTokenScopeCreateDefault(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":          domain.TokenScopeGitRead,
		"  ":        domain.TokenScopeGitRead,
		"read":      domain.TokenScopeGitRead,
		"git:read":  domain.TokenScopeGitRead,
		"write":     domain.TokenScopeGitWrite,
		"git:write": domain.TokenScopeGitWrite,
		"admin":     "",
	}
	for in, want := range cases {
		if got := domain.NormalizeTokenScope(in); got != want {
			t.Errorf("NormalizeTokenScope(%q)=%q want %q", in, got, want)
		}
	}
}

func TestGitReceivePackEmptyScopeForbidden(t *testing.T) {
	t.Parallel()

	uc := &GitUseCase{authz: NewAuthz(nil, nil)}
	repo := &domain.Repository{OwnerID: 1}

	for _, scope := range []string{"", domain.TokenScopeGitRead} {
		err := uc.Authorize(context.Background(), repo, &domain.User{
			ID:         1,
			TokenScope: scope,
		}, GitReceivePack)
		if !errors.Is(err, ErrRepoForbidden) {
			t.Fatalf("scope %q: err=%v want forbidden", scope, err)
		}
	}

	private := &domain.Repository{OwnerID: 1, IsPrivate: true}
	if err := uc.Authorize(context.Background(), private, &domain.User{ID: 1}, GitUploadPack); !errors.Is(err, ErrRepoForbidden) {
		t.Fatalf("empty scope read private: err=%v want forbidden", err)
	}

	if err := uc.Authorize(context.Background(), repo, &domain.User{
		ID:         1,
		TokenScope: domain.TokenScopeGitWrite,
	}, GitReceivePack); err != nil {
		t.Fatalf("git:write owner push: %v", err)
	}
}

func TestCanContributePublicRepoPolicy(t *testing.T) {
	t.Parallel()

	repo := &domain.Repository{ID: 1, OwnerID: 1}
	a := NewAuthz(nil, &memCollabs{})

	if a.CanContribute(repo, 2) {
		t.Fatal("stranger must not contribute to public repo by default")
	}

	if !a.CanContribute(repo, 1) {
		t.Fatal("owner must contribute")
	}

	a.SetAllowPublicContribute(true)
	if !a.CanContribute(repo, 2) {
		t.Fatal("logged-in user may contribute when allow_public_contribute")
	}

	if a.CanContribute(repo, 0) {
		t.Fatal("anonymous must never contribute")
	}

	private := &domain.Repository{ID: 2, OwnerID: 1, IsPrivate: true}
	if a.CanContribute(private, 2) {
		t.Fatal("private repo requires write access")
	}
}

func TestAdminUseCaseRequiresSiteAdmin(t *testing.T) {
	t.Parallel()

	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "admin",
		Email:        "a@x",
		PasswordHash: "h:secret",
		Role:         domain.RoleAdmin,
	},
	)
	_ = users.Create(&domain.User{
		Username:     "user",
		Email:        "u@x",
		PasswordHash: "h:x",
		Role:         domain.RoleUser,
	},
	)

	sessions := &memSessions{}
	uc := NewAdminUseCase(users, sessions, nil, NewAuthUseCase(users, sessions, nil, memHasher{}, nil), memHasher{})
	ctx := context.Background()

	if _, err := uc.Overview(ctx, 2); !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("overview non-admin: %v", err)
	}

	if _, err := uc.ListUsers(ctx, 2, AdminListInput{}); !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("list users non-admin: %v", err)
	}

	if _, err := uc.ListRepos(ctx, 2, domain.AdminRepoFilter{}); !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("list repos non-admin: %v", err)
	}

	if _, err := uc.Create(ctx, 2, AdminCreateInput{
		Role:            domain.RoleUser,
		CurrentPassword: "x",
	}); !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("create non-admin: %v", err)
	}

	if _, err := uc.Update(ctx, 2, 1, AdminUpdateInput{}); !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("update non-admin: %v", err)
	}

	if err := uc.Delete(ctx, 2, 1, "x"); !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("delete non-admin: %v", err)
	}

	if _, err := uc.ListUsers(ctx, 0, AdminListInput{}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("list users anonymous: %v", err)
	}

	if _, err := uc.ListUsers(ctx, 1, AdminListInput{}); err != nil {
		t.Fatalf("list users admin: %v", err)
	}
}

func TestMirrorOwnerOnlyGuardsSyncCheckVerify(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	collabs := &memCollabs{}
	e.uc.authz = NewAuthz(nil, collabs)

	repo := e.addMirror(domain.MirrorPull)
	repo.MirrorOwnerOnly = true
	if err := e.store.Update(repo); err != nil {
		t.Fatal(err)
	}

	_ = collabs.Upsert(&domain.RepoCollaborator{
		RepoID: repo.ID,
		UserID: 2,
		Role:   domain.RepoRoleAdmin,
	})

	ctx := context.Background()
	in := ResolveRepositoryInput{
		Name:     "app",
		ViewerID: 2,
	}

	if _, err := e.uc.SyncMirror(ctx, in); !errors.Is(err, ErrMirrorOwnerOnly) {
		t.Fatalf("sync: %v", err)
	}

	if _, err := e.uc.CheckMirror(ctx, CheckMirrorInput{
		ResolveRepositoryInput: in,
	}); !errors.Is(err, ErrMirrorOwnerOnly) {
		t.Fatalf("check: %v", err)
	}

	if _, err := e.uc.VerifyMirror(ctx, VerifyMirrorInput{
		ResolveRepositoryInput: in,
	}); !errors.Is(err, ErrMirrorOwnerOnly) {
		t.Fatalf("verify: %v", err)
	}

	if _, err := e.uc.ConfigureMirror(ctx, ConfigureMirrorInput{
		ViewerID: 2,
		Name:     "app",
	}); !errors.Is(err, ErrMirrorOwnerOnly) {
		t.Fatalf("configure: %v", err)
	}

	repo.MirrorOwnerOnly = false
	if err := e.store.Update(repo); err != nil {
		t.Fatal(err)
	}

	if _, err := e.uc.SyncMirror(ctx, in); errors.Is(err, ErrMirrorOwnerOnly) {
		t.Fatalf("sync without owner_only: %v", err)
	}
}

func TestHashAccessTokenAlwaysHMAC(t *testing.T) {
	t.Parallel()

	for _, pepper := range []string{"", "pepper"} {
		h := hashAccessToken("repo_x", pepper)
		if !strings.HasPrefix(h, accessTokenHashV2Prefix) || len(h) != 67 {
			t.Fatalf("pepper %q: unexpected hash %q", pepper, h)
		}
	}

	if hashAccessToken("repo_x", "a") == hashAccessToken("repo_x", "b") {
		t.Fatal("hash must depend on pepper")
	}
}

type emptyScopeTokens struct {
	memAccessTokens
}

func (emptyScopeTokens) FindByHash(string) (*domain.AccessToken, error) {
	return &domain.AccessToken{ID: 1, UserID: 1}, nil
}

func TestAuthenticateTokenEmptyScopeUnauthorized(t *testing.T) {
	t.Parallel()

	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username: "u",
		Email:    "u@x",
		Role:     domain.RoleUser,
	})
	uc := &GitUseCase{users: users, tokens: &emptyScopeTokens{}}

	if _, err := uc.authenticateToken(context.Background(), "repo_x"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("empty stored scope: err=%v want unauthorized", err)
	}
}
