package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/magomedcoder/repo/internal/domain"
)

func stepUpEnv(t *testing.T) (*mirrorEnv, *domain.Repository) {
	t.Helper()

	e := newMirrorEnv()
	users, ok := e.uc.users.(*memUsers)
	if !ok {
		t.Fatalf("unexpected users store %T", e.uc.users)
	}

	users.items[0].PasswordHash = "h:secret"
	e.uc.SetPasswordHasher(memHasher{})

	repo := e.addMirror(domain.MirrorPush)

	return e, repo
}

func TestRepositoryUpdateStepUp(t *testing.T) {
	t.Parallel()

	boolPtr := func(b bool) *bool { return &b }
	strPtr := func(s string) *string { return &s }

	cases := []struct {
		name string
		in   UpdateRepositoryInput
		want error
	}{
		{"make private without password", UpdateRepositoryInput{
			Private: boolPtr(true),
		},
			ErrPasswordRequired,
		},
		{"make private wrong password", UpdateRepositoryInput{
			Private:         boolPtr(true),
			CurrentPassword: "bad",
		},
			ErrCurrentPasswordWrong,
		},
		{"make private ok", UpdateRepositoryInput{
			Private:         boolPtr(true),
			CurrentPassword: "secret"},
			nil,
		},
		{"rename without password", UpdateRepositoryInput{
			NewName: strPtr("renamed")},
			ErrPasswordRequired,
		},
		{"rename wrong password", UpdateRepositoryInput{
			NewName:         strPtr("renamed"),
			CurrentPassword: "bad",
		},
			ErrCurrentPasswordWrong,
		},
		{"same visibility needs no password", UpdateRepositoryInput{
			Private: boolPtr(false),
		},
			nil,
		},
		{"same name needs no password", UpdateRepositoryInput{
			NewName: strPtr("app"),
		},
			nil,
		},
		{"description needs no password", UpdateRepositoryInput{
			Description: strPtr("hello"),
		},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			e, _ := stepUpEnv(t)
			in := tc.in
			in.ViewerID, in.Name = 1, "app"

			_, err := e.uc.Update(context.Background(), in)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("err=%v want nil", err)
				}

				return
			}

			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
		})
	}
}

func TestConfigureMirrorStepUp(t *testing.T) {
	t.Parallel()

	strPtr := func(s string) *string { return &s }

	base := func() ConfigureMirrorInput {
		return ConfigureMirrorInput{
			ViewerID:  1,
			Name:      "app",
			Direction: domain.MirrorPush,
			URL:       "https://git.example.com/org/app.git",
			Enabled:   true,
		}
	}

	cases := []struct {
		name   string
		mutate func(*ConfigureMirrorInput)
		want   error
	}{
		{"unchanged url and no secrets", func(*ConfigureMirrorInput) {}, nil},
		{"new url without password", func(in *ConfigureMirrorInput) {
			in.URL = "https://git.example.com/other.git"
		},
			ErrPasswordRequired,
		},
		{"new url wrong password", func(in *ConfigureMirrorInput) {
			in.URL = "https://git.example.com/other.git"
			in.CurrentPassword = "bad"
		},
			ErrCurrentPasswordWrong},
		{"new url ok", func(in *ConfigureMirrorInput) {
			in.URL = "https://git.example.com/other.git"
			in.CurrentPassword = "secret"
		},
			nil,
		},
		{"new password without step-up", func(in *ConfigureMirrorInput) {
			in.Password = strPtr("pw")
		},
			ErrPasswordRequired,
		},
		{"new password ok", func(in *ConfigureMirrorInput) {
			in.Password = strPtr("pw")
			in.CurrentPassword = "secret"
		}, nil,
		},
		{"clear password without step-up", func(in *ConfigureMirrorInput) {
			in.ClearPassword = true
		},
			ErrPasswordRequired,
		},
		{"clear ssh key without step-up", func(in *ConfigureMirrorInput) {
			in.ClearSSHKey = true
		},
			ErrPasswordRequired,
		},
		{"ssh passphrase without step-up", func(in *ConfigureMirrorInput) {
			in.SSHPassphrase = strPtr("p")
		},
			ErrPasswordRequired,
		},
		{"interval only needs no password", func(in *ConfigureMirrorInput) {
			in.Interval = strPtr("1h")
		},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			e, _ := stepUpEnv(t)
			in := base()
			tc.mutate(&in)

			_, err := e.uc.ConfigureMirror(context.Background(), in)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("err=%v want nil", err)
				}

				return
			}

			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
		})
	}
}

type trustGit struct {
	*fakeGit
	configured bool
}

func (g trustGit) VerifyTrustConfigured() bool { return g.configured }

func TestConfigureMirrorVerifySignedRequiresTrust(t *testing.T) {
	t.Parallel()

	yes := true

	for _, tc := range []struct {
		name       string
		configured bool
		want       error
	}{
		{"no trust material", false, ErrMirrorVerifyNoTrust},
		{"trust material loaded", true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			e, _ := stepUpEnv(t)
			e.uc.git = trustGit{
				fakeGit:    e.git,
				configured: tc.configured,
			}

			_, err := e.uc.ConfigureMirror(context.Background(), ConfigureMirrorInput{
				ViewerID:     1,
				Name:         "app",
				Direction:    domain.MirrorPush,
				URL:          "https://git.example.com/org/app.git",
				Enabled:      true,
				VerifySigned: &yes,
			})
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
		})
	}
}

func TestGitAuthorizeRequireAuthRead(t *testing.T) {
	t.Parallel()

	public := &domain.Repository{
		ID:      1,
		OwnerID: 1,
	}
	viewer := &domain.User{
		ID: 2,
	}

	cases := []struct {
		name    string
		require bool
		viewer  *domain.User
		want    error
	}{
		{"default: anonymous may clone public", false, nil, nil},
		{"flag: anonymous refused", true, nil, ErrUnauthorized},
		{"flag: logged-in allowed", true, viewer, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authz := NewAuthz(nil, nil)
			authz.SetRequireAuthRead(tc.require)
			uc := &GitUseCase{authz: authz}

			err := uc.Authorize(context.Background(), public, tc.viewer, GitUploadPack)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
		})
	}
}

func TestTokenCreateRevokeStepUp(t *testing.T) {
	t.Parallel()

	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "u",
		Email:        "u@x",
		PasswordHash: "h:secret",
		Role:         domain.RoleUser,
	})

	store := &memAccessTokens{}
	uc := NewTokenUseCase(store, &memTokens{})
	uc.SetPepper("0123456789abcdef0123456789abcdef")
	uc.SetStepUp(users, memHasher{})
	ctx := context.Background()

	if _, err := uc.Create(ctx, CreateTokenInput{
		UserID: 1,
		Name:   "ci",
	}); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("create no password: %v", err)
	}

	if _, err := uc.Create(ctx, CreateTokenInput{
		UserID:          1,
		Name:            "ci",
		CurrentPassword: "bad",
	}); !errors.Is(err, ErrCurrentPasswordWrong) {
		t.Fatalf("create wrong password: %v", err)
	}

	out, err := uc.Create(ctx, CreateTokenInput{
		UserID:          1,
		Name:            "ci",
		CurrentPassword: "secret",
	})
	if err != nil {
		t.Fatalf("create ok: %v", err)
	}

	if err := uc.Revoke(ctx, 1, out.ID, ""); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("revoke no password: %v", err)
	}

	if err := uc.Revoke(ctx, 1, out.ID, "bad"); !errors.Is(err, ErrCurrentPasswordWrong) {
		t.Fatalf("revoke wrong password: %v", err)
	}

	if err := uc.Revoke(ctx, 1, out.ID, "secret"); err != nil {
		t.Fatalf("revoke ok: %v", err)
	}
}
