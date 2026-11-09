package usecase

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/attempts"
)

type GitService string

const (
	GitUploadPack  GitService = "git-upload-pack"
	GitReceivePack GitService = "git-receive-pack"
)

type GitUseCase struct {
	repos         domain.RepositoryStore
	users         domain.UserStore
	tokens        domain.AccessTokenStore
	hasher        domain.PasswordHasher
	git           domain.GitRepository
	activity      domain.ActivityStore
	authz         *Authz
	guard         *LoginGuard
	tokenAttempts attempts.Counter
	pepper        string
	passwordAuth  bool
}

func NewGitUseCase(
	repos domain.RepositoryStore,
	users domain.UserStore,
	tokens domain.AccessTokenStore,
	hasher domain.PasswordHasher,
	git domain.GitRepository,
	activity domain.ActivityStore,
	authz *Authz,
) *GitUseCase {
	return &GitUseCase{
		repos:    repos,
		users:    users,
		tokens:   tokens,
		hasher:   hasher,
		git:      git,
		activity: activity,
		authz:    authz,
		guard:    NewLoginGuard(),
		tokenAttempts: attempts.New(attempts.Config{
			MaxFails: gitTokenMaxFails,
		}),
		passwordAuth: true,
	}
}

func (uc *GitUseCase) SetTokenPepper(pepper string) {
	if uc != nil {
		uc.pepper = pepper
	}
}

func (uc *GitUseCase) StartAttemptsSweep(ctx context.Context, every time.Duration) {
	attempts.StartBackgroundSweep(ctx, every, uc.tokenAttempts)
}

func (uc *GitUseCase) SetLoginGuard(g *LoginGuard) {
	if uc != nil && g != nil {
		uc.guard = g
	}
}

func (uc *GitUseCase) SetPasswordAuth(v bool) {
	if uc != nil {
		uc.passwordAuth = v
	}
}

func (uc *GitUseCase) AnonymousReadDenied() bool {
	return uc != nil && uc.authz != nil && uc.authz.AnonymousReadDenied()
}

func (uc *GitUseCase) LoginRetryAfter(ctx context.Context, login, ipKey string) time.Duration {
	if uc == nil || uc.guard == nil {
		return 0
	}

	login = normalizeLogin(login)
	var user *domain.User
	if login != "" {
		var err error
		if strings.Contains(login, "@") {
			user, err = uc.users.FindByEmail(ctx, login)
		} else {
			user, err = uc.users.FindByUsername(ctx, login)
		}

		if err != nil {
			user = nil
		}
	}

	return uc.guard.RetryAfter(ipKey, AccountKey(user, login))
}

func (uc *GitUseCase) ResolveClonePath(ctx context.Context, path string) (*domain.Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return nil, ErrRepoNotFound
	}
	if !strings.HasSuffix(path, ".git") {
		path += ".git"
	}

	parts := strings.Split(path, "/")
	if len(parts) != domain.MaxPathDepth || parts[0] == "" {
		return nil, ErrRepoNotFound
	}

	if IsReservedSlug(strings.TrimSuffix(parts[0], ".git")) {
		return nil, ErrRepoNotFound
	}

	repo, err := uc.repos.FindByRepoPath(ctx, path)
	if err != nil {
		return nil, ErrRepoNotFound
	}

	return repo, nil
}

func (uc *GitUseCase) Authenticate(ctx context.Context, remoteIP, username, secret string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	username = normalizeLogin(username)
	secret = strings.TrimSpace(secret)

	if strings.HasPrefix(secret, "repo_") {
		user, err := uc.authenticateTokenWithAttempts(ctx, remoteIP, secret)
		if err != nil {
			return nil, err
		}

		if username != "" && user.Username != username {
			return nil, ErrUnauthorized
		}

		return user, nil
	}

	if !uc.passwordAuth {
		return nil, ErrUnauthorized
	}

	if uc.guard.IPLocked(remoteIP) {
		return nil, ErrTooManyAttempts
	}

	if username == "" {
		return nil, ErrUnauthorized
	}

	var (
		user *domain.User
		err  error
	)
	if strings.Contains(username, "@") {
		user, err = uc.users.FindByEmail(ctx, username)
	} else {
		user, err = uc.users.FindByUsername(ctx, username)
	}

	if err != nil {
		user = nil
	}

	acctKey := AccountKey(user, username)
	if uc.guard.AccountLocked(acctKey) {
		return nil, ErrTooManyAttempts
	}

	if secret == "" || len(secret) > maxPasswordLen {
		return nil, ErrUnauthorized
	}

	if user == nil {
		if err := uc.hasher.Compare(dummyPasswordHash, secret); isHasherBusy(err) {
			return nil, ErrBusy
		}

		uc.guard.Fail(remoteIP, acctKey)
		return nil, ErrUnauthorized
	}

	if err := uc.hasher.Compare(user.PasswordHash, secret); err != nil {
		if isHasherBusy(err) {
			return nil, ErrBusy
		}

		uc.guard.Fail(remoteIP, acctKey)
		return nil, ErrUnauthorized
	}

	uc.guard.Success(acctKey)

	out := sanitizeUser(user)
	out.TokenScope = domain.TokenScopeGitWrite

	return out, nil
}

func (uc *GitUseCase) authenticateTokenWithAttempts(ctx context.Context, remoteIP, raw string) (*domain.User, error) {
	attemptKey := "git-token\n" + remoteIP
	if remoteIP != "" && uc.tokenAttempts != nil && uc.tokenAttempts.Locked(attemptKey) {
		return nil, ErrTooManyAttempts
	}

	user, err := uc.authenticateToken(ctx, raw)
	if err != nil {
		if remoteIP != "" && uc.tokenAttempts != nil {
			_ = uc.tokenAttempts.Fail(attemptKey)
		}

		return nil, err
	}

	return user, nil
}

func (uc *GitUseCase) authenticateToken(ctx context.Context, raw string) (*domain.User, error) {
	token, err := uc.tokens.FindByHash(hashAccessToken(raw, uc.pepper))
	if err != nil {
		return nil, ErrUnauthorized
	}

	if AccessTokenExpired(token, time.Now()) {
		return nil, ErrUnauthorized
	}

	if token.Scope != domain.TokenScopeGitRead && token.Scope != domain.TokenScopeGitWrite {
		return nil, ErrUnauthorized
	}

	user, err := uc.users.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	_ = uc.tokens.TouchLastUsed(token.ID, time.Now())
	out := sanitizeUser(user)
	out.TokenScope = token.Scope
	out.TokenRepoIDs = token.RepoIDs

	return out, nil
}

func (uc *GitUseCase) Authorize(ctx context.Context, repo *domain.Repository, viewer *domain.User, service GitService) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	switch service {
	case GitUploadPack:
		if !repo.IsPrivate {
			if uc.authz != nil && uc.authz.AnonymousReadDenied() && viewer == nil {
				return ErrUnauthorized
			}

			return nil
		}

		if viewer == nil {
			return ErrUnauthorized
		}

		if viewer.TokenScope == domain.TokenScopeGitRead || viewer.TokenScope == domain.TokenScopeGitWrite {
			if !uc.authz.CanReadPrivateRepo(repo, viewer.ID) {
				return ErrRepoForbidden
			}

			if !tokenAllowsRepo(viewer, repo) {
				return ErrRepoForbidden
			}

			return nil
		}

		return ErrRepoForbidden
	case GitReceivePack:
		if viewer == nil {
			return ErrUnauthorized
		}

		if viewer.TokenScope != domain.TokenScopeGitWrite {
			return ErrRepoForbidden
		}

		if !uc.authz.CanWriteGit(repo, viewer.ID) {
			return ErrRepoForbidden
		}

		if !tokenAllowsRepo(viewer, repo) {
			return ErrRepoForbidden
		}

		if repo.MirrorDirection == domain.MirrorPull {
			return ErrMirrorReadOnly
		}

		return nil
	default:
		return fmt.Errorf("unsupported git service: %s", service)
	}
}

func tokenAllowsRepo(viewer *domain.User, repo *domain.Repository) bool {
	if viewer == nil || len(viewer.TokenRepoIDs) == 0 {
		return true
	}

	return slices.Contains(viewer.TokenRepoIDs, repo.ID)
}

func (uc *GitUseCase) AdvertiseRefs(ctx context.Context, repoPath string, service GitService, w io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return uc.git.AdvertiseRefs(ctx, repoPath, string(service), w)
}

func (uc *GitUseCase) ServePack(ctx context.Context, repoPath string, service GitService, stdin io.Reader, stdout io.Writer, receive *domain.ReceivePackOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return uc.git.ServePack(ctx, repoPath, string(service), stdin, stdout, receive)
}

func (uc *GitUseCase) LockRepo(repoPath string) func() {
	return uc.git.LockRepo(repoPath)
}

func (uc *GitUseCase) EnsureDefaultBranch(ctx context.Context, repoPath, branch string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return uc.git.EnsureDefaultBranch(repoPath, branch)
}

func (uc *GitUseCase) TouchActivity(ctx context.Context, repo *domain.Repository, actorID uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if repo == nil {
		return nil
	}

	_ = uc.repos.TouchLastActivity(repo.ID, time.Now())

	recordActivity(uc.activity, &domain.ActivityEvent{
		ActorID: actorID,
		OwnerID: repo.OwnerID,
		RepoID:  &repo.ID,
		Type:    domain.ActivityPush,
		Summary: "pushed to repository",
	})

	return nil
}

func ParseGitService(raw string) (GitService, error) {
	switch strings.TrimSpace(raw) {
	case "git-upload-pack", "upload-pack":
		return GitUploadPack, nil
	case "git-receive-pack", "receive-pack":
		return GitReceivePack, nil
	default:
		return "", fmt.Errorf("unsupported service")
	}
}
