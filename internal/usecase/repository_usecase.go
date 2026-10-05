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
	ErrNameRequired      = errors.New("repository_name_required")
	ErrAlreadyExists     = errors.New("repository_already_exists")
	ErrInvalidName       = errors.New("invalid_repository_name")
	ErrRepoNotFound      = errors.New("repository_not_found")
	ErrRepoForbidden     = errors.New("repository_access_denied")
	ErrInvalidBranchName = errors.New("invalid_default_branch")
)

var repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type CreateRepositoryInput struct {
	Name          string
	Description   string
	ViewerID      uint
	OwnerID       uint
	Organization  string
	FolderID      *uint
	Private       bool
	DefaultBranch string
	BasePath      string
}

type UpdateRepositoryInput struct {
	ViewerID      uint
	OwnerUsername string
	FolderPath    string
	Name          string
	Description   *string
	Private       *bool
	DefaultBranch *string
}

type MoveRepositoryInput struct {
	ViewerID      uint
	OwnerUsername string
	FolderPath    string
	Name          string
	NewFolderID   *uint
	BasePath      string
}

type ResolveRepositoryInput struct {
	OwnerUsername string
	FolderPath    string
	Name          string
	ViewerID      uint
}

type RepositoryItem struct {
	ID             uint       `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Owner          string     `json:"owner"`
	FolderID       *uint      `json:"folder_id"`
	FolderPath     string     `json:"folder_path"`
	IsPrivate      bool       `json:"is_private"`
	DefaultBranch  string     `json:"default_branch"`
	LastActivityAt *time.Time `json:"last_activity_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type RepositoryUseCase struct {
	store   domain.RepositoryStore
	folders domain.FolderStore
	users   domain.UserStore
	orgs    domain.OrganizationStore
	git     domain.GitRepository
}

func NewRepositoryUseCase(
	store domain.RepositoryStore,
	folders domain.FolderStore,
	users domain.UserStore,
	orgs domain.OrganizationStore,
	git domain.GitRepository,
) *RepositoryUseCase {
	return &RepositoryUseCase{
		store:   store,
		folders: folders,
		users:   users,
		orgs:    orgs,
		git:     git,
	}
}

func (uc *RepositoryUseCase) Create(in CreateRepositoryInput) (*RepositoryItem, error) {
	viewerID := in.ViewerID
	if viewerID == 0 {
		viewerID = in.OwnerID
	}

	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, err
	}

	ownerKind := domain.OwnerKindUser
	ownerID := in.OwnerID
	ownerSlug := ""
	if strings.TrimSpace(in.Organization) != "" {
		org, err := uc.orgs.FindBySlug(strings.ToLower(strings.TrimSpace(in.Organization)))
		if err != nil {
			return nil, ErrOrgNotFound
		}

		member, err := uc.orgs.FindMember(org.ID, viewerID)
		if err != nil || !orgRoleAtLeast(member.Role, domain.OrgRoleAdmin) {
			return nil, ErrOrgForbidden
		}

		ownerKind = domain.OwnerKindOrg
		ownerID = org.ID
		ownerSlug = org.Slug
	} else {
		if ownerID == 0 {
			ownerID = viewerID
		}

		if ownerID != viewerID {
			return nil, ErrRepoForbidden
		}

		owner, err := uc.users.FindByID(ownerID)
		if err != nil {
			return nil, ErrUnauthorized
		}

		ownerSlug = owner.Username
	}

	folderPath, err := uc.resolveFolderForOwner(ownerKind, ownerID, in.FolderID)
	if err != nil {
		return nil, err
	}

	exists, err := uc.store.ExistsByOwnerFolderName(ownerKind, ownerID, in.FolderID, name)
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
	repoPath := buildRepoDiskPath(basePath, ownerKind, ownerID, folderPath, name)

	if err := uc.git.InitBare(repoPath, domain.BareInitOptions{
		DefaultBranch:        branch,
		DenyForcePushDefault: true,
	}); err != nil {
		return nil, fmt.Errorf("init bare repository: %w", err)
	}

	repo := &domain.Repository{
		Name:          name,
		OwnerKind:     ownerKind,
		OwnerID:       ownerID,
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

	item := toRepositoryItem(repo, ownerSlug, folderPath)
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

	repos, err := uc.store.ListByOwnerID(domain.OwnerKindUser, ownerID)
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

func (uc *RepositoryUseCase) Search(query string, ownerID uint, publicOnly bool) ([]RepositoryItem, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		if publicOnly || ownerID == 0 {
			return uc.ListPublic()
		}
		return uc.ListOwn(ownerID)
	}

	repos, err := uc.store.SearchByName(q, domain.OwnerKindUser, ownerID, publicOnly || ownerID == 0, 100)
	if err != nil {
		return nil, fmt.Errorf("search repositories: %w", err)
	}

	if publicOnly || ownerID == 0 {
		return uc.mapReposWithOwners(repos)
	}

	owner, err := uc.users.FindByID(ownerID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return uc.mapRepos(repos, owner.Username)
}

func (uc *RepositoryUseCase) Get(in ResolveRepositoryInput) (*RepositoryItem, error) {
	repo, ownerSlug, folderPath, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if err := uc.authorizeRead(repo, in.ViewerID); err != nil {
		return nil, err
	}

	item := toRepositoryItem(repo, ownerSlug, folderPath)
	return &item, nil
}

func (uc *RepositoryUseCase) Update(in UpdateRepositoryInput) (*RepositoryItem, error) {
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, folderPath, err := uc.resolve(ResolveRepositoryInput{
		OwnerUsername: in.OwnerUsername,
		FolderPath:    in.FolderPath,
		Name:          in.Name,
		ViewerID:      in.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	if !uc.canAdminRepo(repo, in.ViewerID) {
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

	item := toRepositoryItem(repo, ownerSlug, folderPath)
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

	if !uc.canAdminRepo(repo, in.ViewerID) {
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
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, _, err := uc.resolve(ResolveRepositoryInput{
		OwnerUsername: in.OwnerUsername,
		FolderPath:    in.FolderPath,
		Name:          in.Name,
		ViewerID:      in.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	if !uc.canAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoForbidden
	}

	sameFolder := (repo.FolderID == nil && in.NewFolderID == nil) || (repo.FolderID != nil && in.NewFolderID != nil && *repo.FolderID == *in.NewFolderID)
	if sameFolder {
		folderPath, _ := uc.resolveFolderForOwner(repo.OwnerKind, repo.OwnerID, repo.FolderID)
		item := toRepositoryItem(repo, ownerSlug, folderPath)
		return &item, nil
	}

	newFolderPath, err := uc.resolveFolderForOwner(repo.OwnerKind, repo.OwnerID, in.NewFolderID)
	if err != nil {
		return nil, err
	}

	exists, err := uc.store.ExistsByOwnerFolderName(repo.OwnerKind, repo.OwnerID, in.NewFolderID, repo.Name)
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
	newPath := buildRepoDiskPath(basePath, repo.OwnerKind, repo.OwnerID, newFolderPath, repo.Name)

	if err := uc.git.Move(oldPath, newPath); err != nil {
		return nil, fmt.Errorf("move repository files: %w", err)
	}

	repo.FolderID = in.NewFolderID
	repo.Path = newPath
	if err := uc.store.Update(repo); err != nil {
		_ = uc.git.Move(newPath, oldPath)
		return nil, fmt.Errorf("update repository metadata: %w", err)
	}

	item := toRepositoryItem(repo, ownerSlug, newFolderPath)
	return &item, nil
}

func (uc *RepositoryUseCase) resolve(in ResolveRepositoryInput) (*domain.Repository, string, string, error) {
	ownerSlug := strings.ToLower(strings.TrimSpace(in.OwnerUsername))
	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, "", "", err
	}

	ownerKind, ownerID, err := uc.resolveNamespace(ownerSlug)
	if err != nil {
		return nil, "", "", err
	}

	folderPath := strings.Trim(strings.TrimSpace(in.FolderPath), "/")
	var folderID *uint
	if folderPath != "" {
		folder, err := uc.folders.FindByOwnerAndPath(ownerKind, ownerID, folderPath)
		if err != nil {
			return nil, "", "", ErrRepoNotFound
		}
		folderID = &folder.ID
	}

	repo, err := uc.store.FindByOwnerFolderName(ownerKind, ownerID, folderID, name)
	if err != nil {
		return nil, "", "", ErrRepoNotFound
	}

	return repo, ownerSlug, folderPath, nil
}

func (uc *RepositoryUseCase) resolveNamespace(slug string) (kind string, id uint, err error) {
	if user, uerr := uc.users.FindByUsername(slug); uerr == nil {
		return domain.OwnerKindUser, user.ID, nil
	}

	if uc.orgs != nil {
		if org, oerr := uc.orgs.FindBySlug(slug); oerr == nil {
			return domain.OwnerKindOrg, org.ID, nil
		}
	}

	return "", 0, ErrRepoNotFound
}

func (uc *RepositoryUseCase) authorizeRead(repo *domain.Repository, viewerID uint) error {
	if !repo.IsPrivate {
		return nil
	}

	if uc.canReadPrivate(repo, viewerID) {
		return nil
	}

	return ErrRepoForbidden
}

func (uc *RepositoryUseCase) canReadPrivate(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 {
		return false
	}

	kind := repo.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	if kind == domain.OwnerKindUser {
		return viewerID == repo.OwnerID
	}

	if uc.orgs == nil {
		return false
	}

	_, err := uc.orgs.FindMember(repo.OwnerID, viewerID)
	return err == nil
}

func (uc *RepositoryUseCase) canAdminRepo(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 {
		return false
	}

	kind := repo.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	if kind == domain.OwnerKindUser {
		return viewerID == repo.OwnerID
	}

	if uc.orgs == nil {
		return false
	}

	member, err := uc.orgs.FindMember(repo.OwnerID, viewerID)
	if err != nil {
		return false
	}

	return orgRoleAtLeast(member.Role, domain.OrgRoleAdmin)
}

func (uc *RepositoryUseCase) canWriteGit(repo *domain.Repository, viewerID uint) bool {
	return uc.canAdminRepo(repo, viewerID)
}

func (uc *RepositoryUseCase) resolveFolderForOwner(ownerKind string, ownerID uint, folderID *uint) (string, error) {
	if folderID == nil {
		return "", nil
	}

	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	folder, err := uc.folders.FindByOwnerAndID(ownerKind, ownerID, *folderID)
	if err != nil {
		return "", ErrFolderNotFound
	}

	return folder.Path, nil
}

func (uc *RepositoryUseCase) mapRepos(repos []domain.Repository, ownerUsername string) ([]RepositoryItem, error) {
	items := make([]RepositoryItem, 0, len(repos))
	for _, repo := range repos {
		folderPath := ""
		kind := repo.OwnerKind
		if kind == "" {
			kind = domain.OwnerKindUser
		}

		if repo.FolderID != nil {
			folder, err := uc.folders.FindByOwnerAndID(kind, repo.OwnerID, *repo.FolderID)
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
	userCache := map[uint]string{}
	orgCache := map[uint]string{}
	for _, repo := range repos {
		kind := repo.OwnerKind
		if kind == "" {
			kind = domain.OwnerKindUser
		}

		ownerSlug := ""
		if kind == domain.OwnerKindOrg {
			slug, ok := orgCache[repo.OwnerID]
			if !ok {
				if uc.orgs == nil {
					continue
				}

				org, err := uc.orgs.FindByID(repo.OwnerID)
				if err != nil {
					continue
				}

				slug = org.Slug
				orgCache[repo.OwnerID] = slug
			}
			ownerSlug = slug
		} else {
			username, ok := userCache[repo.OwnerID]
			if !ok {
				user, err := uc.users.FindByID(repo.OwnerID)
				if err != nil {
					continue
				}

				username = user.Username
				userCache[repo.OwnerID] = username
			}
			ownerSlug = username
		}

		folderPath := ""
		if repo.FolderID != nil {
			folder, err := uc.folders.FindByOwnerAndID(kind, repo.OwnerID, *repo.FolderID)
			if err == nil {
				folderPath = folder.Path
			}
		}
		items = append(items, toRepositoryItem(&repo, ownerSlug, folderPath))
	}

	return items, nil
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
		ID:             repo.ID,
		Name:           repo.Name,
		Description:    repo.Description,
		Owner:          ownerUsername,
		FolderID:       repo.FolderID,
		FolderPath:     folderPath,
		IsPrivate:      repo.IsPrivate,
		DefaultBranch:  repo.DefaultBranch,
		LastActivityAt: repo.LastActivityAt,
		CreatedAt:      repo.CreatedAt,
		UpdatedAt:      repo.UpdatedAt,
	}
}
