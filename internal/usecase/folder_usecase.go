package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"magomedcoder.ru/repo/internal/domain"
)

const (
	maxFolderNameLen = 64
	minFolderNameLen = 1
)

var (
	ErrFolderNameRequired = NewError(KindValidation, "folder_name_required")
	ErrInvalidFolderName  = NewError(KindValidation, "invalid_folder_name")
	ErrFolderExists       = NewError(KindConflict, "folder_already_exists")
	ErrFolderNotFound     = NewError(KindNotFound, "folder_not_found")
	ErrFolderNotEmpty     = NewError(KindConflict, "folder_not_empty")
	ErrFolderForbidden    = NewError(KindForbidden, "folder_access_denied")
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9_-]+`)

type FolderItem struct {
	ID        uint            `json:"-"`
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Path      string          `json:"path"`
	Role      string          `json:"role,omitzero"`
	Caps      map[string]bool `json:"caps,omitzero"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type FolderContents struct {
	Folder *FolderItem             `json:"folder,omitzero"`
	Repos  []RepositorySummaryItem `json:"repos"`
}

type RepositorySummaryItem struct {
	ID            uint      `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	IsPrivate     bool      `json:"is_private"`
	DefaultBranch string    `json:"default_branch"`
	RepoPath      string    `json:"repo_path"`
	FolderPath    string    `json:"folder_path,omitzero"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateFolderInput struct {
	ViewerID uint
	OwnerID  uint
	Name     string
}

type RenameFolderInput struct {
	ViewerID uint
	OwnerID  uint
	Path     string
	Name     string
}

type FolderUseCase struct {
	folders domain.FolderStore
	repos   domain.RepositoryStore
	members domain.FolderMemberStore
	git     domain.GitRepository
	authz   *Authz
}

func NewFolderUseCase(folders domain.FolderStore, repos domain.RepositoryStore, members domain.FolderMemberStore, git domain.GitRepository, authz *Authz) *FolderUseCase {
	return &FolderUseCase{
		folders: folders,
		repos:   repos,
		members: members,
		git:     git,
		authz:   authz,
	}
}

func (uc *FolderUseCase) Create(ctx context.Context, in CreateFolderInput) (*FolderItem, error) {
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

	name, slug, err := normalizeFolderName(in.Name)
	if err != nil {
		return nil, err
	}

	if IsReservedSlug(slug) {
		return nil, ErrReservedSlug
	}

	exists, err := uc.folders.ExistsBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("check folder existence: %w", err)
	}

	if exists {
		return nil, ErrFolderExists
	}

	folder := &domain.Folder{
		OwnerKind: domain.OwnerKindUser,
		OwnerID:   viewerID,
		Name:      name,
		Slug:      slug,
		Path:      slug,
	}
	if err := uc.folders.Create(folder); err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}

	if uc.members != nil {
		member := &domain.FolderMember{
			FolderID: folder.ID,
			UserID:   viewerID,
			Role:     domain.FolderRoleOwner,
			Caps:     domain.CapsForRole(domain.FolderRoleOwner),
		}

		if err := uc.members.Create(member); err != nil {
			_ = uc.folders.Delete(folder.ID)
			return nil, fmt.Errorf("seed folder owner: %w", err)
		}
	}

	item := toFolderItem(folder)
	item.Role = string(domain.FolderRoleOwner)
	item.Caps = CapsJSON(domain.CapsForRole(domain.FolderRoleOwner))

	return &item, nil
}

func (uc *FolderUseCase) List(ctx context.Context, viewerID, _ uint) ([]FolderItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	folders, err := uc.folders.ListForMember(viewerID)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}

	items := make([]FolderItem, 0, len(folders))
	for i := range folders {
		item := toFolderItem(&folders[i])
		if uc.authz != nil {
			caps, role, cerr := uc.authz.FolderCaps(folders[i].ID, viewerID)
			if cerr == nil {
				item.Role = string(role)
				item.Caps = CapsJSON(caps)
			}
		}
		items = append(items, item)
	}

	return items, nil
}

func (uc *FolderUseCase) GetContents(ctx context.Context, viewerID, _ uint, path string) (*FolderContents, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	path = strings.Trim(path, "/")
	if path == "" {
		return nil, ErrFolderNotFound
	}

	folder, err := uc.folders.FindByPath(path)
	if err != nil {
		return nil, ErrFolderNotFound
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapView); err != nil {
		return nil, ErrFolderNotFound
	}

	return uc.folderContents(viewerID, folder)
}

func (uc *FolderUseCase) folderContents(viewerID uint, folder *domain.Folder) (*FolderContents, error) {
	repos, err := uc.repos.ListByFolderID(context.Background(), folder.ID)
	if err != nil {
		return nil, fmt.Errorf("list folder repos: %w", err)
	}

	folderItem := toFolderItem(folder)
	if uc.authz != nil {
		caps, role, cerr := uc.authz.FolderCaps(folder.ID, viewerID)
		if cerr == nil {
			folderItem.Role = string(role)
			folderItem.Caps = CapsJSON(caps)
		}
	}

	out := &FolderContents{
		Folder: &folderItem,
		Repos:  make([]RepositorySummaryItem, 0, len(repos)),
	}

	for _, repo := range repos {
		if uc.authz != nil {
			if err := uc.authz.CanReadRepo(&repo, viewerID); err != nil {
				continue
			}
		} else if repo.IsPrivate && !isPersonalOwner(&repo, viewerID) {
			continue
		}

		out.Repos = append(out.Repos, toRepoSummary(repo, folder.Path))
	}

	return out, nil
}

func (uc *FolderUseCase) Rename(ctx context.Context, in RenameFolderInput) (*FolderItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	viewerID := in.ViewerID
	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	path := strings.Trim(in.Path, "/")
	if path == "" {
		return nil, ErrFolderNotFound
	}

	folder, err := uc.folders.FindByPath(path)
	if err != nil {
		return nil, ErrFolderNotFound
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapManageSettings); err != nil {
		return nil, err
	}

	name, slug, err := normalizeFolderName(in.Name)
	if err != nil {
		return nil, err
	}

	if slug != folder.Slug {
		if IsReservedSlug(slug) {
			return nil, ErrReservedSlug
		}
		exists, err := uc.folders.ExistsBySlug(slug)
		if err != nil {
			return nil, fmt.Errorf("check folder existence: %w", err)
		}
		if exists {
			return nil, ErrFolderExists
		}
	}

	oldPath := folder.Path
	folder.Name = name
	folder.Slug = slug
	folder.Path = slug

	var moved []folderMovedRepo

	if oldPath != slug {
		repos, err := uc.repos.ListByFolderID(ctx, folder.ID)
		if err != nil {
			return nil, fmt.Errorf("list folder repos: %w", err)
		}

		for i := range repos {
			repo := &repos[i]
			oldDisk := repo.Path
			oldLogic := repo.RepoPath
			newLogical := slug + "/" + repo.Name + ".git"
			newDisk := buildRepoDiskPath(uc.git.GetReposRoot(), repo.OwnerKind, repo.OwnerID, slug, repo.Name)
			if uc.git != nil && repo.Path != "" && repo.Path != newDisk {
				if err := uc.git.Move(repo.Path, newDisk); err != nil {
					uc.rollbackFolderMoves(moved)
					return nil, fmt.Errorf("move repository files: %w", err)
				}
			}

			repo.Path = newDisk
			repo.RepoPath = newLogical
			if err := uc.repos.Update(repo); err != nil {
				uc.rollbackFolderMoves(append(moved, folderMovedRepo{repo: repo, oldDisk: oldDisk, oldLogic: oldLogic}))
				return nil, fmt.Errorf("update repository path: %w", err)
			}
			moved = append(moved, folderMovedRepo{repo: repo, oldDisk: oldDisk, oldLogic: oldLogic})
		}
	}

	if err := uc.folders.Update(folder); err != nil {
		uc.rollbackFolderMoves(moved)
		return nil, fmt.Errorf("rename folder: %w", err)
	}

	item := toFolderItem(folder)
	return &item, nil
}

func (uc *FolderUseCase) Delete(ctx context.Context, viewerID, _ uint, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if viewerID == 0 {
		return ErrUnauthorized
	}

	path = strings.Trim(path, "/")
	if path == "" {
		return ErrFolderNotFound
	}

	folder, err := uc.folders.FindByPath(path)
	if err != nil {
		return ErrFolderNotFound
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapDeleteFolder); err != nil {
		return err
	}

	childRepos, err := uc.repos.CountByFolderID(folder.ID)
	if err != nil {
		return fmt.Errorf("count folder repos: %w", err)
	}

	if childRepos > 0 {
		return ErrFolderNotEmpty
	}

	if err := uc.folders.Delete(folder.ID); err != nil {
		return fmt.Errorf("delete folder: %w", err)
	}

	return nil
}

func (uc *FolderUseCase) FindByPath(path string) (*domain.Folder, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil, ErrFolderNotFound
	}

	folder, err := uc.folders.FindByPath(path)
	if err != nil {
		return nil, ErrFolderNotFound
	}

	return folder, nil
}

func normalizeFolderName(raw string) (name, slug string, err error) {
	name = strings.TrimSpace(raw)
	if name == "" {
		return "", "", ErrFolderNameRequired
	}

	if len(name) < minFolderNameLen || len([]rune(name)) > maxFolderNameLen {
		return "", "", ErrInvalidFolderName
	}

	for _, r := range name {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return "", "", ErrInvalidFolderName
		}
	}

	slug = strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonSlugChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-_")
	if slug == "" || slug == "." || slug == ".." {
		return "", "", ErrInvalidFolderName
	}

	if !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`).MatchString(slug) {
		return "", "", ErrInvalidFolderName
	}

	return name, slug, nil
}

func toFolderItem(f *domain.Folder) FolderItem {
	return FolderItem{
		ID:        f.ID,
		Name:      f.Name,
		Slug:      f.Slug,
		Path:      f.Path,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

type folderMovedRepo struct {
	repo     *domain.Repository
	oldDisk  string
	oldLogic string
}

func (uc *FolderUseCase) rollbackFolderMoves(moved []folderMovedRepo) {
	var errs []error
	for _, m := range slices.Backward(moved) {
		if uc.git != nil && m.repo.Path != "" && m.oldDisk != "" && m.repo.Path != m.oldDisk {
			if err := uc.git.Move(m.repo.Path, m.oldDisk); err != nil {
				errs = append(errs, fmt.Errorf("rollback git move %s: %w", m.repo.Path, err))
			}
		}

		m.repo.Path = m.oldDisk
		m.repo.RepoPath = m.oldLogic
		if err := uc.repos.Update(m.repo); err != nil {
			errs = append(errs, fmt.Errorf("rollback repo metadata %d: %w", m.repo.ID, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		slog.Error("folder rollback", "err", err)
	}
}

func toRepoSummary(repo domain.Repository, folderPath string) RepositorySummaryItem {
	repoPath := strings.TrimSuffix(repo.RepoPath, ".git")
	if repoPath == "" {
		if folderPath != "" {
			repoPath = folderPath + "/" + repo.Name
		} else {
			repoPath = repo.Name
		}
	}

	return RepositorySummaryItem{
		ID:            repo.ID,
		Name:          repo.Name,
		Description:   repo.Description,
		IsPrivate:     repo.IsPrivate,
		DefaultBranch: repo.DefaultBranch,
		RepoPath:      repoPath,
		FolderPath:    folderPath,
		CreatedAt:     repo.CreatedAt,
		UpdatedAt:     repo.UpdatedAt,
	}
}
