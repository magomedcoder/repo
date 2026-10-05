package usecase

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type GitService string

const (
	GitUploadPack  GitService = "git-upload-pack"
	GitReceivePack GitService = "git-receive-pack"
)

type GitUseCase struct {
	repos   domain.RepositoryStore
	folders domain.FolderStore
	users   domain.UserStore
	orgs    domain.OrganizationStore
	tokens  domain.AccessTokenStore
	hasher  domain.PasswordHasher
	git     domain.GitRepository
}

func NewGitUseCase(
	repos domain.RepositoryStore,
	folders domain.FolderStore,
	users domain.UserStore,
	orgs domain.OrganizationStore,
	tokens domain.AccessTokenStore,
	hasher domain.PasswordHasher,
	git domain.GitRepository,
) *GitUseCase {
	return &GitUseCase{
		repos:   repos,
		folders: folders,
		users:   users,
		orgs:    orgs,
		tokens:  tokens,
		hasher:  hasher,
		git:     git,
	}
}

func (uc *GitUseCase) Resolve(ownerUsername, folderPath, name string) (*domain.Repository, error) {
	repo, _, _, err := (&RepositoryUseCase{
		store:   uc.repos,
		folders: uc.folders,
		users:   uc.users,
		orgs:    uc.orgs,
	}).resolve(ResolveRepositoryInput{
		OwnerUsername: ownerUsername,
		FolderPath:    folderPath,
		Name:          name,
	})
	return repo, err
}

func (uc *GitUseCase) Authenticate(username, secret string) (*domain.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	secret = strings.TrimSpace(secret)

	if strings.HasPrefix(username, "repo_") {
		return uc.authenticateToken(username)
	}

	if strings.HasPrefix(secret, "repo_") {
		user, err := uc.authenticateToken(secret)
		if err != nil {
			return nil, ErrUnauthorized
		}

		if username != "" && user.Username != username {
			return nil, ErrUnauthorized
		}

		return user, nil
	}

	if username == "" || secret == "" {
		return nil, ErrUnauthorized
	}

	user, err := uc.users.FindByUsername(username)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if err := uc.hasher.Compare(user.PasswordHash, secret); err != nil {
		return nil, ErrUnauthorized
	}

	return sanitizeUser(user), nil
}

func (uc *GitUseCase) authenticateToken(raw string) (*domain.User, error) {
	token, err := uc.tokens.FindByHash(hashAccessToken(raw))
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := uc.users.FindByID(token.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	_ = uc.tokens.TouchLastUsed(token.ID, time.Now())
	return sanitizeUser(user), nil
}

func (uc *GitUseCase) Authorize(repo *domain.Repository, viewer *domain.User, service GitService) error {
	repoUC := &RepositoryUseCase{store: uc.repos, folders: uc.folders, users: uc.users, orgs: uc.orgs}
	switch service {
	case GitUploadPack:
		if !repo.IsPrivate {
			return nil
		}

		if viewer == nil {
			return ErrUnauthorized
		}

		if !repoUC.canReadPrivate(repo, viewer.ID) {
			return ErrRepoForbidden
		}

		return nil
	case GitReceivePack:
		if viewer == nil {
			return ErrUnauthorized
		}

		if !repoUC.canWriteGit(repo, viewer.ID) {
			return ErrRepoForbidden
		}
		
		return nil
	default:
		return fmt.Errorf("unsupported git service: %s", service)
	}
}

func (uc *GitUseCase) AdvertiseRefs(repoPath string, service GitService, w io.Writer) error {
	return uc.git.AdvertiseRefs(repoPath, string(service), w)
}

func (uc *GitUseCase) ServePack(repoPath string, service GitService, stdin io.Reader, stdout io.Writer) error {
	return uc.git.ServePack(repoPath, string(service), stdin, stdout)
}

func (uc *GitUseCase) ServeSSHPack(repoPath string, service GitService, stdin io.Reader, stdout, stderr io.Writer) error {
	return uc.git.ServeSSHPack(repoPath, string(service), stdin, stdout, stderr)
}

func (uc *GitUseCase) TouchActivity(repoID uint) error {
	return uc.repos.TouchLastActivity(repoID, time.Now())
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
