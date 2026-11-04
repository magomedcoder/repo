package usecase

import (
	"context"
	"errors"
	"fmt"
	"magomedcoder.ru/repo/internal/config"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v6/plumbing"

	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrNameRequired      = NewError(KindValidation, "repository_name_required")
	ErrAlreadyExists     = NewError(KindConflict, "repository_already_exists")
	ErrInvalidName       = NewError(KindValidation, "invalid_repository_name")
	ErrRepoNotFound      = NewError(KindNotFound, "repository_not_found")
	ErrRepoForbidden     = NewError(KindForbidden, "repository_access_denied")
	ErrInvalidBranchName = NewError(KindValidation, "invalid_default_branch")
	ErrRepoQuota         = NewError(KindForbidden, "repository_quota_exceeded")
)

const maxReposPerOwner = 100

var repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}

type CreateRepositoryInput struct {
	Name          string
	Path          string
	Description   string
	ViewerID      uint
	OwnerID       uint
	FolderPath    string
	Private       bool
	ShowOnHome    bool
	DefaultBranch string
}

type UpdateRepositoryInput struct {
	ViewerID             uint
	FolderPath           string
	Name                 string
	NewName              *string
	Description          *string
	Private              *bool
	ShowOnHome           *bool
	DefaultBranch        *string
	ProtectDefaultBranch *bool
	IssuesEnabled        *bool
}

type MoveRepositoryInput struct {
	ViewerID      uint
	FolderPath    string
	Name          string
	NewFolderPath string
}

type ResolveRepositoryInput struct {
	FolderPath string
	Name       string
	ViewerID   uint
}

type RepositoryItem struct {
	ID                   uint       `json:"id"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	Owner                string     `json:"owner"`
	FolderID             *uint      `json:"-"`
	FolderPath           string     `json:"folder_path"`
	IsPrivate            bool       `json:"is_private"`
	ShowOnHome           bool       `json:"show_on_home"`
	ProtectDefaultBranch bool       `json:"protect_default_branch"`
	IssuesEnabled        bool       `json:"issues_enabled"`
	DefaultBranch        string     `json:"default_branch"`
	LastActivityAt       *time.Time `json:"last_activity_at,omitzero"`
	RepoPath             string     `json:"repo_path"`
	CanRead              bool       `json:"can_read"`
	CanWrite             bool       `json:"can_write"`
	CanAdmin             bool       `json:"can_admin"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type RepositoryUseCase struct {
	store    domain.RepositoryStore
	folders  domain.FolderStore
	users    domain.UserStore
	git      domain.GitRepository
	activity domain.ActivityStore
	authz    *Authz
}

func NewRepositoryUseCase(
	store domain.RepositoryStore,
	folders domain.FolderStore,
	users domain.UserStore,
	git domain.GitRepository,
	activity domain.ActivityStore,
	authz *Authz,
) *RepositoryUseCase {
	return &RepositoryUseCase{
		store:    store,
		folders:  folders,
		users:    users,
		git:      git,
		activity: activity,
		authz:    authz,
	}
}

func (uc *RepositoryUseCase) Create(ctx context.Context, in CreateRepositoryInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	viewerID := in.ViewerID
	if viewerID == 0 {
		viewerID = in.OwnerID
	}

	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	name := ""
	if strings.TrimSpace(in.Path) == "" {
		var nerr error
		name, nerr = normalizeRepoName(in.Name)
		if nerr != nil {
			return nil, nerr
		}
	} else {
		name = strings.TrimSpace(in.Name)
	}

	ownerKind := domain.OwnerKindUser
	ownerID := viewerID

	owner, err := uc.users.FindByID(ctx, ownerID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	ownerSlug := owner.Username

	if owner.Role != domain.RoleAdmin {
		n, cerr := uc.store.CountByOwnerID(ownerKind, ownerID)
		if cerr != nil {
			return nil, fmt.Errorf("count repositories: %w", cerr)
		}

		if n >= maxReposPerOwner {
			return nil, ErrRepoQuota
		}
	}

	folderID, folderPath, err := uc.resolveFolderPathForCreate(viewerID, in.FolderPath)
	if err != nil {
		return nil, err
	}

	logical := name + ".git"
	if folderPath != "" {
		logical = folderPath + "/" + name + ".git"
	}

	if strings.TrimSpace(in.Path) != "" {
		parsed, perr := domain.ParseRepoPath(in.Path)
		if perr != nil {
			return nil, ErrInvalidName
		}

		if len(parsed) > 2 {
			return nil, ErrInvalidName
		}

		name = strings.TrimSuffix(parsed[len(parsed)-1], ".git")
		folderID = nil
		folderPath = ""
		if len(parsed) == 2 {
			folderID, folderPath, err = uc.resolveFolderPathForCreate(viewerID, parsed[0])
			if err != nil {
				return nil, err
			}
		}
		logical = name + ".git"
		if folderPath != "" {
			logical = folderPath + "/" + name + ".git"
		}
	}

	if _, err := domain.ParseRepoPath(logical); err != nil {
		return nil, ErrInvalidName
	}

	firstSegment := name
	if folderPath != "" {
		firstSegment = strings.Split(folderPath, "/")[0]
	}
	if IsReservedSlug(firstSegment) {
		return nil, ErrReservedSlug
	}

	exists, err := uc.store.ExistsByOwnerFolderName(ownerKind, ownerID, folderID, name)
	if err != nil {
		return nil, fmt.Errorf("check repository existence: %w", err)
	}

	if exists {
		return nil, ErrAlreadyExists
	}

	if other, ferr := uc.store.FindByRepoPath(ctx, logical); ferr == nil && other != nil {
		return nil, ErrAlreadyExists
	}

	branch := strings.TrimSpace(in.DefaultBranch)
	if branch == "" {
		branch = config.DefaultBranch
	}

	if err := validateBranchName(branch); err != nil {
		return nil, err
	}

	repoPath := buildRepoDiskPath(uc.git.GetReposRoot(), ownerKind, ownerID, folderPath, name)

	repo := &domain.Repository{
		Name:                 name,
		OwnerKind:            ownerKind,
		OwnerID:              ownerID,
		FolderID:             folderID,
		RepoPath:             logical,
		Description:          strings.TrimSpace(in.Description),
		IsPrivate:            in.Private,
		ShowOnHome:           !in.Private && in.ShowOnHome,
		ProtectDefaultBranch: true,
		IssuesEnabled:        true,
		DefaultBranch:        branch,
		Path:                 repoPath,
	}
	if err := uc.store.Create(repo); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}

		return nil, fmt.Errorf("save repository metadata: %w", err)
	}

	if err := uc.git.InitBare(repoPath, domain.BareInitOptions{
		DefaultBranch: branch,
	}); err != nil {
		_ = uc.store.Delete(repo.ID)
		return nil, fmt.Errorf("init bare repository: %w", err)
	}

	item := uc.toRepositoryItem(repo, ownerSlug, folderPath, viewerID)
	recordActivity(uc.activity, &domain.ActivityEvent{
		ActorID:   viewerID,
		OwnerKind: ownerKind,
		OwnerID:   ownerID,
		RepoID:    &repo.ID,
		Type:      domain.ActivityRepoCreate,
		Summary:   activitySummary("created repository %s", name),
	})

	return &item, nil
}

func (uc *RepositoryUseCase) ListOwn(ctx context.Context, ownerID uint, limit, offset int, order domain.RepoOrder) ([]RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if ownerID == 0 {
		return nil, ErrUnauthorized
	}

	owner, err := uc.users.FindByID(ctx, ownerID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	offset, limit = NormalizePagination(offset, limit)
	repos, err := uc.store.ListByOwnerID(ctx, domain.OwnerKindUser, ownerID, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	return uc.mapRepos(repos, owner.Username)
}

func (uc *RepositoryUseCase) ListPublic(ctx context.Context, limit, offset int, order domain.RepoOrder) ([]RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	offset, limit = NormalizePagination(offset, limit)
	repos, err := uc.store.ListPublic(ctx, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("list public repositories: %w", err)
	}

	return uc.mapReposWithOwners(repos)
}

func (uc *RepositoryUseCase) ListHome(ctx context.Context, limit, offset int, order domain.RepoOrder) ([]RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	offset, limit = NormalizePagination(offset, limit)
	repos, err := uc.store.ListHome(ctx, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("list home repositories: %w", err)
	}

	return uc.mapReposWithOwners(repos)
}

func (uc *RepositoryUseCase) Search(ctx context.Context, query string, ownerID uint, publicOnly bool, limit, offset int, order domain.RepoOrder) ([]RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}

	offset, limit = NormalizePagination(offset, limit)
	repos, err := uc.store.SearchByName(ctx, q, domain.OwnerKindUser, ownerID, publicOnly || ownerID == 0, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("search repositories: %w", err)
	}

	if publicOnly || ownerID == 0 {
		return uc.mapReposWithOwners(repos)
	}

	owner, err := uc.users.FindByID(ctx, ownerID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return uc.mapRepos(repos, owner.Username)
}

func (uc *RepositoryUseCase) Get(ctx context.Context, in ResolveRepositoryInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, ownerSlug, folderPath, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	item := uc.toRepositoryItem(repo, ownerSlug, folderPath, in.ViewerID)
	return &item, nil
}

func (uc *RepositoryUseCase) Update(ctx context.Context, in UpdateRepositoryInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, folderPath, err := uc.resolve(ResolveRepositoryInput{
		FolderPath: in.FolderPath,
		Name:       in.Name,
		ViewerID:   in.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	if in.Description != nil {
		repo.Description = strings.TrimSpace(*in.Description)
	}

	if in.Private != nil {
		repo.IsPrivate = *in.Private
	}

	if in.ShowOnHome != nil {
		repo.ShowOnHome = *in.ShowOnHome
	}

	if repo.IsPrivate {
		repo.ShowOnHome = false
	}

	if in.DefaultBranch != nil {
		branch := strings.TrimSpace(*in.DefaultBranch)
		if err := validateBranchName(branch); err != nil {
			return nil, err
		}

		refs, berr := uc.git.ListBranches(repo.Path)
		if berr != nil {
			return nil, fmt.Errorf("list branches: %w", berr)
		}

		if len(refs) > 0 {
			found := false
			for _, ref := range refs {
				if ref.Name == branch {
					found = true
					break
				}
			}

			if !found {
				return nil, ErrInvalidBranchName
			}
		}

		repo.DefaultBranch = branch
	}

	if in.ProtectDefaultBranch != nil {
		repo.ProtectDefaultBranch = *in.ProtectDefaultBranch
	}

	if in.IssuesEnabled != nil {
		repo.IssuesEnabled = *in.IssuesEnabled
	}

	oldDisk := repo.Path
	renamed := false
	if in.NewName != nil {
		newName, nerr := normalizeRepoName(*in.NewName)
		if nerr != nil {
			return nil, nerr
		}

		if newName != repo.Name {
			if folderPath == "" && IsReservedSlug(newName) {
				return nil, ErrReservedSlug
			}

			exists, eerr := uc.store.ExistsByOwnerFolderName(repo.OwnerKind, repo.OwnerID, repo.FolderID, newName)
			if eerr != nil {
				return nil, fmt.Errorf("check repository existence: %w", eerr)
			}

			if exists {
				return nil, ErrAlreadyExists
			}

			newLogical := newName + ".git"
			if folderPath != "" {
				newLogical = folderPath + "/" + newName + ".git"
			}

			if other, ferr := uc.store.FindByRepoPath(ctx, newLogical); ferr == nil && other.ID != repo.ID {
				return nil, ErrAlreadyExists
			}

			newDisk := buildRepoDiskPath(uc.git.GetReposRoot(), repo.OwnerKind, repo.OwnerID, folderPath, newName)
			if err := uc.git.Move(oldDisk, newDisk); err != nil {
				return nil, fmt.Errorf("rename repository files: %w", err)
			}

			repo.Name = newName
			repo.Path = newDisk
			repo.RepoPath = newLogical
			renamed = true
		}
	}

	if err := uc.store.Update(repo); err != nil {
		if renamed {
			_ = uc.git.Move(repo.Path, oldDisk)
		}
		return nil, fmt.Errorf("update repository: %w", err)
	}

	item := uc.toRepositoryItem(repo, ownerSlug, folderPath, in.ViewerID)
	return &item, nil
}

func (uc *RepositoryUseCase) Delete(ctx context.Context, in ResolveRepositoryInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if in.ViewerID == 0 {
		return ErrUnauthorized
	}

	repo, _, _, err := uc.resolve(in)
	if err != nil {
		return err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return ErrRepoNotFound
	}

	diskPath := repo.Path
	metaErr := uc.store.Delete(repo.ID)
	diskErr := uc.git.Remove(diskPath)
	if metaErr != nil {
		metaErr = fmt.Errorf("delete repository metadata: %w", metaErr)
	}
	if diskErr != nil {
		diskErr = fmt.Errorf("delete repository files: %w", diskErr)
	}
	return errors.Join(metaErr, diskErr)
}

func (uc *RepositoryUseCase) Move(ctx context.Context, in MoveRepositoryInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, _, err := uc.resolve(ResolveRepositoryInput{
		FolderPath: in.FolderPath,
		Name:       in.Name,
		ViewerID:   in.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	newFolderID, newFolderPath, err := uc.resolveFolderPathForMove(in.ViewerID, in.NewFolderPath)
	if err != nil {
		return nil, err
	}

	sameFolder := (repo.FolderID == nil && newFolderID == nil) ||
		(repo.FolderID != nil && newFolderID != nil && *repo.FolderID == *newFolderID)
	if sameFolder {
		currentPath, _ := uc.resolveFolderPathByID(repo.FolderID)
		item := uc.toRepositoryItem(repo, ownerSlug, currentPath, in.ViewerID)
		return &item, nil
	}

	if newFolderPath == "" {
		if IsReservedSlug(repo.Name) {
			return nil, ErrReservedSlug
		}
	}

	exists, err := uc.store.ExistsByOwnerFolderName(repo.OwnerKind, repo.OwnerID, newFolderID, repo.Name)
	if err != nil {
		return nil, fmt.Errorf("check repository existence: %w", err)
	}

	if exists {
		return nil, ErrAlreadyExists
	}

	newLogical := repo.Name + ".git"
	if newFolderPath != "" {
		newLogical = newFolderPath + "/" + repo.Name + ".git"
	}
	if other, ferr := uc.store.FindByRepoPath(ctx, newLogical); ferr == nil && other.ID != repo.ID {
		return nil, ErrAlreadyExists
	}

	oldPath := repo.Path
	newPath := buildRepoDiskPath(uc.git.GetReposRoot(), repo.OwnerKind, repo.OwnerID, newFolderPath, repo.Name)

	if err := uc.git.Move(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("move repository files: %w", err)
	}

	repo.FolderID = newFolderID
	repo.Path = newPath
	repo.RepoPath = newLogical
	if err := uc.store.Update(repo); err != nil {
		_ = uc.git.Move(newPath, oldPath)
		return nil, fmt.Errorf("update repository metadata: %w", err)
	}

	item := uc.toRepositoryItem(repo, ownerSlug, newFolderPath, in.ViewerID)
	return &item, nil
}

func (uc *RepositoryUseCase) resolve(in ResolveRepositoryInput) (*domain.Repository, string, string, error) {
	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, "", "", err
	}

	folderPath := strings.Trim(strings.TrimSpace(in.FolderPath), "/")
	logical := name + ".git"
	if folderPath != "" {
		logical = folderPath + "/" + name + ".git"
	}

	if _, err := domain.ParseRepoPath(logical); err != nil {
		return nil, "", "", ErrRepoNotFound
	}

	repo, err := uc.store.FindByRepoPath(context.Background(), logical)
	if err != nil {
		return nil, "", "", ErrRepoNotFound
	}

	ownerSlug := ""
	if user, uerr := uc.users.FindByID(context.Background(), repo.OwnerID); uerr == nil {
		ownerSlug = user.Username
	}

	return repo, ownerSlug, folderPath, nil
}

func (uc *RepositoryUseCase) resolveFolderPathByID(folderID *uint) (string, error) {
	if folderID == nil {
		return "", nil
	}

	folder, err := uc.folders.FindByID(*folderID)
	if err != nil {
		return "", ErrFolderNotFound
	}

	return folder.Path, nil
}

func (uc *RepositoryUseCase) resolveFolderPathForCreate(viewerID uint, path string) (*uint, string, error) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return nil, "", nil
	}

	folder, err := uc.folders.FindByPath(path)
	if err != nil {
		return nil, "", ErrFolderNotFound
	}

	if !uc.authz.HasFolderCap(folder.ID, viewerID, domain.CapCreateRepos) {
		return nil, "", ErrFolderForbidden
	}

	id := folder.ID
	return &id, folder.Path, nil
}

func (uc *RepositoryUseCase) resolveFolderPathForMove(viewerID uint, path string) (*uint, string, error) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return nil, "", nil
	}

	folder, err := uc.folders.FindByPath(path)
	if err != nil {
		return nil, "", ErrFolderNotFound
	}

	if !uc.authz.HasFolderCap(folder.ID, viewerID, domain.CapCreateRepos) {
		return nil, "", ErrFolderForbidden
	}

	id := folder.ID
	return &id, folder.Path, nil
}

func (uc *RepositoryUseCase) mapRepos(repos []domain.Repository, ownerUsername string) ([]RepositoryItem, error) {
	folderCache := uc.folderPathCache(repos)
	items := make([]RepositoryItem, 0, len(repos))
	for _, repo := range repos {
		folderPath := ""
		if repo.FolderID != nil {
			folderPath = folderCache[*repo.FolderID]
		}
		items = append(items, uc.toRepositoryItem(&repo, ownerUsername, folderPath, repo.OwnerID))
	}
	return items, nil
}

func (uc *RepositoryUseCase) mapReposWithOwners(repos []domain.Repository) ([]RepositoryItem, error) {
	folderCache := uc.folderPathCache(repos)
	items := make([]RepositoryItem, 0, len(repos))
	userCache := map[uint]string{}
	for _, repo := range repos {
		kind := repo.OwnerKind
		if kind == "" {
			kind = domain.OwnerKindUser
		}
		if kind != domain.OwnerKindUser {
			continue
		}

		username, ok := userCache[repo.OwnerID]
		if !ok {
			user, err := uc.users.FindByID(context.Background(), repo.OwnerID)
			if err != nil {
				continue
			}
			username = user.Username
			userCache[repo.OwnerID] = username
		}

		folderPath := ""
		if repo.FolderID != nil {
			folderPath = folderCache[*repo.FolderID]
		}
		items = append(items, uc.toRepositoryItem(&repo, username, folderPath, 0))
	}
	return items, nil
}

func (uc *RepositoryUseCase) folderPathCache(repos []domain.Repository) map[uint]string {
	cache := map[uint]string{}
	for _, repo := range repos {
		if repo.FolderID == nil {
			continue
		}

		id := *repo.FolderID
		if _, ok := cache[id]; ok {
			continue
		}

		folder, err := uc.folders.FindByID(id)
		if err == nil {
			cache[id] = folder.Path
		} else {
			cache[id] = ""
		}
	}

	return cache
}

func buildRepoDiskPath(basePath, ownerKind string, ownerID uint, folderPath, name string) string {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	parts := []string{basePath, ownerKind, fmt.Sprintf("%d", ownerID)}
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
	if name == "" || plumbing.ValidateBranchName(name) != nil {
		return ErrInvalidBranchName
	}

	return nil
}

func (uc *RepositoryUseCase) toRepositoryItem(repo *domain.Repository, ownerUsername, folderPath string, viewerID uint) RepositoryItem {
	item := RepositoryItem{
		ID:                   repo.ID,
		Name:                 repo.Name,
		Description:          repo.Description,
		Owner:                ownerUsername,
		FolderID:             repo.FolderID,
		FolderPath:           folderPath,
		IsPrivate:            repo.IsPrivate,
		ShowOnHome:           repo.ShowOnHome && !repo.IsPrivate,
		ProtectDefaultBranch: repo.ProtectDefaultBranch,
		IssuesEnabled:        repo.IssuesEnabled,
		DefaultBranch:        repo.DefaultBranch,
		LastActivityAt:       repo.LastActivityAt,
		RepoPath:             repo.RepoPath,
		CreatedAt:            repo.CreatedAt,
		UpdatedAt:            repo.UpdatedAt,
	}
	if viewerID != 0 && uc.authz != nil {
		item.CanRead = uc.authz.CanReadRepo(repo, viewerID) == nil
		item.CanWrite = uc.authz.CanWriteRepo(repo, viewerID)
		item.CanAdmin = uc.authz.CanAdminRepo(repo, viewerID)
	} else if !repo.IsPrivate {
		item.CanRead = true
	}

	return item
}
