package usecase

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/attempts"
)

type GitService string

const (
	GitUploadPack  GitService = "git-upload-pack"
	GitReceivePack GitService = "git-receive-pack"
)

type GitUseCase struct {
	repos    domain.RepositoryStore
	users    domain.UserStore
	tokens   domain.AccessTokenStore
	hasher   domain.PasswordHasher
	git      domain.GitRepository
	activity domain.ActivityStore
	authz    *Authz
	attempts attempts.Counter
	pepper   string
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
		attempts: attempts.New(),
	}
}

func (uc *GitUseCase) SetTokenPepper(pepper string) {
	if uc != nil {
		uc.pepper = pepper
	}
}

func (uc *GitUseCase) StartAttemptsSweep(ctx context.Context, every time.Duration) {
	attempts.StartBackgroundSweep(ctx, every, uc.attempts)
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

	username = strings.ToLower(strings.TrimSpace(username))
	secret = strings.TrimSpace(secret)

	if strings.HasPrefix(secret, "repo_") {
		user, err := uc.authenticateTokenWithAttempts(secret)
		if err != nil {
			return nil, err
		}

		if username != "" && user.Username != username {
			return nil, ErrUnauthorized
		}

		return user, nil
	}

	if username == "" || secret == "" {
		return nil, ErrUnauthorized
	}

	attemptKey := "git\n" + remoteIP + "\n" + username
	if uc.attempts != nil && uc.attempts.Locked(attemptKey) {
		return nil, ErrTooManyAttempts
	}

	user, err := uc.users.FindByUsername(ctx, username)
	if err != nil {
		_ = uc.hasher.Compare(dummyPasswordHash, secret)
		if uc.attempts != nil {
			_ = uc.attempts.Fail(attemptKey)
		}
		return nil, ErrUnauthorized
	}

	if err := uc.hasher.Compare(user.PasswordHash, secret); err != nil {
		if uc.attempts != nil {
			_ = uc.attempts.Fail(attemptKey)
		}
		return nil, ErrUnauthorized
	}

	if uc.attempts != nil {
		uc.attempts.Reset(attemptKey)
	}

	return sanitizeUser(user), nil
}

func (uc *GitUseCase) authenticateTokenWithAttempts(raw string) (*domain.User, error) {
	attemptKey := "git-token\n" + hashAccessToken(raw, uc.pepper)
	if uc.attempts != nil && uc.attempts.Locked(attemptKey) {
		return nil, ErrTooManyAttempts
	}

	user, err := uc.authenticateToken(raw)
	if err != nil {
		if uc.attempts != nil {
			_ = uc.attempts.Fail(attemptKey)
		}
		return nil, err
	}

	if uc.attempts != nil {
		uc.attempts.Reset(attemptKey)
	}

	return user, nil
}

func (uc *GitUseCase) authenticateToken(raw string) (*domain.User, error) {
	token, err := lookupAccessTokenHash(uc.tokens, raw, uc.pepper)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if AccessTokenExpired(token, time.Now()) {
		return nil, ErrUnauthorized
	}

	user, err := uc.users.FindByID(context.Background(), token.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	_ = uc.tokens.TouchLastUsed(token.ID, time.Now())
	out := sanitizeUser(user)
	scope := token.Scope
	if scope == "" {
		scope = domain.TokenScopeGitWrite
	}
	out.TokenScope = scope

	return out, nil
}

func (uc *GitUseCase) Authorize(ctx context.Context, repo *domain.Repository, viewer *domain.User, service GitService) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	switch service {
	case GitUploadPack:
		if !repo.IsPrivate {
			return nil
		}

		if viewer == nil {
			return ErrUnauthorized
		}

		if viewer.TokenScope == domain.TokenScopeGitRead || viewer.TokenScope == domain.TokenScopeGitWrite || viewer.TokenScope == "" {
			if !uc.authz.CanReadPrivateRepo(repo, viewer.ID) {
				return ErrRepoForbidden
			}
			return nil
		}

		return ErrRepoForbidden
	case GitReceivePack:
		if repo.MirrorDirection == domain.MirrorPull {
			return ErrMirrorReadOnly
		}

		if viewer == nil {
			return ErrUnauthorized
		}

		if viewer.TokenScope == domain.TokenScopeGitRead {
			return ErrRepoForbidden
		}

		if !uc.authz.CanWriteGit(repo, viewer.ID) {
			return ErrRepoForbidden
		}

		return nil
	default:
		return fmt.Errorf("unsupported git service: %s", service)
	}
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
