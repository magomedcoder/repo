package usecase

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrNameRequired      = errors.New("repository name required")
	ErrAlreadyExists     = errors.New("repository already exists")
	ErrInvalidName       = errors.New("invalid repository name")
	ErrRepoNotFound      = errors.New("repository not found")
	ErrRepoForbidden     = errors.New("repository access denied")
	ErrInvalidBranchName = errors.New("invalid default branch")
)

var repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type CreateRepositoryInput struct {
	Name          string
	Description   string
	OwnerID       uint
	FolderID      *uint
	Private       bool
	DefaultBranch string
	BasePath      string
}

type UpdateRepositoryInput struct {
	OwnerID       uint
	OwnerUsername string
	FolderPath    string // logical folder path; empty = root
	Name          string
	Description   *string
	Private       *bool
	DefaultBranch *string
}

type MoveRepositoryInput struct {
	OwnerID       uint
	OwnerUsername string
	FolderPath    string
	Name          string
	NewFolderID   *uint
	BasePath      string
}

type ResolveRepositoryInput struct {
	OwnerUsername string
	FolderPath    string // "" = root; "work/backend"
	Name          string
	ViewerID      uint // 0 = anonymous
}

type RepositoryItem struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Owner         string    `json:"owner"`
	FolderID      *uint     `json:"folder_id"`
	FolderPath    string    `json:"folder_path"`
	IsPrivate     bool      `json:"is_private"`
	DefaultBranch string    `json:"default_branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RepositoryUseCase struct {
	store   domain.RepositoryStore
	folders domain.FolderStore
	users   domain.UserStore
	git     domain.GitRepository
}

func NewRepositoryUseCase(
	store domain.RepositoryStore,
	folders domain.FolderStore,
	users domain.UserStore,
	git domain.GitRepository,
) *RepositoryUseCase {
	return &RepositoryUseCase{
		store:   store,
		folders: folders,
		users:   users,
		git:     git,
	}
}

func (uc *RepositoryUseCase) Create(in CreateRepositoryInput) (*RepositoryItem, error) {
	if in.OwnerID == 0 {
		return nil, ErrUnauthorized
	}

	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, err
	}

	owner, err := uc.users.FindByID(in.OwnerID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	folderPath, err := uc.resolveFolderForOwner(in.OwnerID, in.FolderID)
	if err != nil {
		return nil, err
	}

	exists, err := uc.store.ExistsByOwnerFolderName(in.OwnerID, in.FolderID, name)
	if err != nil {
		return nil, fmt.Errorf("check repository existence: %w", err)
	}
	if exists {
		return nil, ErrAlreadyExists
	}

	branch := strings.TrimSpace(in.DefaultBranch)
	if branch == "" {
		branch = "main"
	}
	if err := validateBranchName(branch); err != nil {
		return nil, err
	}

	basePath := in.BasePath
	if basePath == "" {
		basePath = filepath.Join("data", "repos")
	}
	repoPath := buildRepoDiskPath(basePath, in.OwnerID, folderPath, name)

	if err := uc.git.InitBare(repoPath); err != nil {
		return nil, fmt.Errorf("init bare repository: %w", err)
	}

	repo := &domain.Repository{
		Name:          name,
		OwnerID:       in.OwnerID,
		FolderID:      in.FolderID,
		Description:   strings.TrimSpace(in.Description),
		IsPrivate:     in.Private,
		DefaultBranch: branch,
		Path:          repoPath,
	}
	if err := uc.store.Create(repo); err != nil {
		_ = uc.git.Remove(repoPath)
		return nil, fmt.Errorf("save repository metadata: %w", err)
	}

	item := toRepositoryItem(repo, owner.Username, folderPath)
	return &item, nil
}

func (uc *RepositoryUseCase) ListOwn(ownerID uint) ([]RepositoryItem, error) {
	if ownerID == 0 {
		return nil, ErrUnauthorized
	}

	owner, err := uc.users.FindByID(ownerID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	repos, err := uc.store.ListByOwnerID(ownerID)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	return uc.mapRepos(repos, owner.Username)
}

func (uc *RepositoryUseCase) ListPublic() ([]RepositoryItem, error) {
	repos, err := uc.store.ListPublic(100)
	if err != nil {
		return nil, fmt.Errorf("list public repositories: %w", err)
	}

	return uc.mapReposWithOwners(repos)
}

func (uc *RepositoryUseCase) Get(in ResolveRepositoryInput) (*RepositoryItem, error) {
	repo, owner, folderPath, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if err := uc.authorizeRead(repo, in.ViewerID); err != nil {
		return nil, err
	}

	item := toRepositoryItem(repo, owner.Username, folderPath)
	return &item, nil
}

func (uc *RepositoryUseCase) Update(in UpdateRepositoryInput) (*RepositoryItem, error) {
	if in.OwnerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, owner, folderPath, err := uc.resolve(ResolveRepositoryInput{
		OwnerUsername: in.OwnerUsername,
		FolderPath:    in.FolderPath,
		Name:          in.Name,
		ViewerID:      in.OwnerID,
	})
	if err != nil {
		return nil, err
	}

	if repo.OwnerID != in.OwnerID {
		return nil, ErrRepoForbidden
	}

	if in.Description != nil {
		repo.Description = strings.TrimSpace(*in.Description)
	}

	if in.Private != nil {
		repo.IsPrivate = *in.Private
	}

	if in.DefaultBranch != nil {
		branch := strings.TrimSpace(*in.DefaultBranch)
		if err := validateBranchName(branch); err != nil {
			return nil, err
		}
		repo.DefaultBranch = branch
	}

	if err := uc.store.Update(repo); err != nil {
		return nil, fmt.Errorf("update repository: %w", err)
	}

	item := toRepositoryItem(repo, owner.Username, folderPath)
	return &item, nil
}

func (uc *RepositoryUseCase) Delete(in ResolveRepositoryInput) error {
	if in.ViewerID == 0 {
		return ErrUnauthorized
	}

	repo, _, _, err := uc.resolve(in)
	if err != nil {
		return err
	}

	if repo.OwnerID != in.ViewerID {
		return ErrRepoForbidden
	}

	diskPath := repo.Path
	if err := uc.store.Delete(repo.ID); err != nil {
		return fmt.Errorf("delete repository metadata: %w", err)
	}

	if err := uc.git.Remove(diskPath); err != nil {
		return fmt.Errorf("delete repository files: %w", err)
	}

	return nil
}

func (uc *RepositoryUseCase) Move(in MoveRepositoryInput) (*RepositoryItem, error) {
	if in.OwnerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, owner, _, err := uc.resolve(ResolveRepositoryInput{
		OwnerUsername: in.OwnerUsername,
		FolderPath:    in.FolderPath,
		Name:          in.Name,
		ViewerID:      in.OwnerID,
	})
	if err != nil {
		return nil, err
	}

	if repo.OwnerID != in.OwnerID {
		return nil, ErrRepoForbidden
	}

	sameFolder := (repo.FolderID == nil && in.NewFolderID == nil) || (repo.FolderID != nil && in.NewFolderID != nil && *repo.FolderID == *in.NewFolderID)
	if sameFolder {
		folderPath, _ := uc.resolveFolderForOwner(in.OwnerID, repo.FolderID)
		item := toRepositoryItem(repo, owner.Username, folderPath)
		return &item, nil
	}

	newFolderPath, err := uc.resolveFolderForOwner(in.OwnerID, in.NewFolderID)
	if err != nil {
		return nil, err
	}

	exists, err := uc.store.ExistsByOwnerFolderName(in.OwnerID, in.NewFolderID, repo.Name)
	if err != nil {
		return nil, fmt.Errorf("check repository existence: %w", err)
	}

	if exists {
		return nil, ErrAlreadyExists
	}

	basePath := in.BasePath
	if basePath == "" {
		basePath = filepath.Join("data", "repos")
	}

	oldPath := repo.Path
	newPath := buildRepoDiskPath(basePath, in.OwnerID, newFolderPath, repo.Name)

	if err := uc.git.Move(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("move repository files: %w", err)
	}

	repo.FolderID = in.NewFolderID
	repo.Path = newPath
	if err := uc.store.Update(repo); err != nil {
		_ = uc.git.Move(newPath, oldPath)
		return nil, fmt.Errorf("update repository metadata: %w", err)
	}

	item := toRepositoryItem(repo, owner.Username, newFolderPath)
	return &item, nil
}

func (uc *RepositoryUseCase) resolve(in ResolveRepositoryInput) (*domain.Repository, *domain.User, string, error) {
	ownerUsername := strings.ToLower(strings.TrimSpace(in.OwnerUsername))
	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, nil, "", err
	}

	owner, err := uc.users.FindByUsername(ownerUsername)
	if err != nil {
		return nil, nil, "", ErrRepoNotFound
	}

	folderPath := strings.Trim(strings.TrimSpace(in.FolderPath), "/")
	var folderID *uint
	if folderPath != "" {
		folder, err := uc.folders.FindByOwnerAndPath(owner.ID, folderPath)
		if err != nil {
			return nil, nil, "", ErrRepoNotFound
		}

		folderID = &folder.ID
	}

	repo, err := uc.store.FindByOwnerFolderName(owner.ID, folderID, name)
	if err != nil {
		return nil, nil, "", ErrRepoNotFound
	}

	return repo, owner, folderPath, nil
}

func (uc *RepositoryUseCase) authorizeRead(repo *domain.Repository, viewerID uint) error {
	if !repo.IsPrivate {
		return nil
	}

	if viewerID != 0 && viewerID == repo.OwnerID {
		return nil
	}

	return ErrRepoForbidden
}

func (uc *RepositoryUseCase) resolveFolderForOwner(ownerID uint, folderID *uint) (string, error) {
	if folderID == nil {
		return "", nil
	}

	folder, err := uc.folders.FindByOwnerAndID(ownerID, *folderID)
	if err != nil {
		return "", ErrFolderNotFound
	}

	return folder.Path, nil
}

func (uc *RepositoryUseCase) mapRepos(repos []domain.Repository, ownerUsername string) ([]RepositoryItem, error) {
	items := make([]RepositoryItem, 0, len(repos))
	for _, repo := range repos {
		folderPath := ""
		if repo.FolderID != nil {
			folder, err := uc.folders.FindByOwnerAndID(repo.OwnerID, *repo.FolderID)
			if err == nil {
				folderPath = folder.Path
			}
		}

		items = append(items, toRepositoryItem(&repo, ownerUsername, folderPath))
	}

	return items, nil
}

func (uc *RepositoryUseCase) mapReposWithOwners(repos []domain.Repository) ([]RepositoryItem, error) {
	items := make([]RepositoryItem, 0, len(repos))
	ownerCache := map[uint]string{}
	for _, repo := range repos {
		username, ok := ownerCache[repo.OwnerID]
		if !ok {
			user, err := uc.users.FindByID(repo.OwnerID)
			if err != nil {
				continue
			}

			username = user.Username
			ownerCache[repo.OwnerID] = username
		}

		folderPath := ""
		if repo.FolderID != nil {
			folder, err := uc.folders.FindByOwnerAndID(repo.OwnerID, *repo.FolderID)
			if err == nil {
				folderPath = folder.Path
			}
		}
		items = append(items, toRepositoryItem(&repo, username, folderPath))
	}

	return items, nil
}

func buildRepoDiskPath(basePath string, ownerID uint, folderPath, name string) string {
	parts := []string{basePath, fmt.Sprintf("%d", ownerID)}
	if folderPath != "" {
		parts = append(parts, strings.Split(folderPath, "/")...)
	}

	parts = append(parts, name+".git")
	return filepath.Join(parts...)
}

func normalizeRepoName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", ErrNameRequired
	}

	if strings.HasSuffix(strings.ToLower(name), ".git") {
		name = name[:len(name)-4]
	}

	if len(name) < 1 || len(name) > 100 {
		return "", ErrInvalidName
	}

	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return "", ErrInvalidName
	}

	if !repoNamePattern.MatchString(name) {
		return "", ErrInvalidName
	}

	return name, nil
}

func validateBranchName(name string) error {
	if name == "" || strings.ContainsAny(name, " \t\n\\") || strings.HasPrefix(name, "-") {
		return ErrInvalidBranchName
	}

	if strings.Contains(name, "..") || strings.Contains(name, "//") {
		return ErrInvalidBranchName
	}

	return nil
}

func toRepositoryItem(repo *domain.Repository, ownerUsername, folderPath string) RepositoryItem {
	return RepositoryItem{
		ID:            repo.ID,
		Name:          repo.Name,
		Description:   repo.Description,
		Owner:         ownerUsername,
		FolderID:      repo.FolderID,
		FolderPath:    folderPath,
		IsPrivate:     repo.IsPrivate,
		DefaultBranch: repo.DefaultBranch,
		CreatedAt:     repo.CreatedAt,
		UpdatedAt:     repo.UpdatedAt,
	}
}
