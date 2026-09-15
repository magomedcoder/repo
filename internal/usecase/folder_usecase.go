package usecase

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/magomedcoder/repo/internal/domain"
)

const (
	maxFolderDepth   = 5
	maxFolderNameLen = 64
	minFolderNameLen = 1
)

var (
	ErrFolderNameRequired   = errors.New("folder name required")
	ErrInvalidFolderName    = errors.New("invalid folder name")
	ErrFolderExists         = errors.New("folder already exists")
	ErrFolderNotFound       = errors.New("folder not found")
	ErrFolderNotEmpty       = errors.New("folder is not empty")
	ErrFolderDepthExceeded  = errors.New("folder depth exceeded")
	ErrInvalidFolderMove    = errors.New("invalid folder move")
	ErrFolderCycle          = errors.New("cannot move folder into its descendant")
	ErrParentFolderNotFound = errors.New("parent folder not found")
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9_-]+`)

type FolderItem struct {
	ID        uint      `json:"id"`
	ParentID  *uint     `json:"parent_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FolderTreeNode struct {
	FolderItem
	Children []FolderTreeNode `json:"children"`
}

type FolderContents struct {
	Folder  *FolderItem             `json:"folder,omitempty"`
	Folders []FolderItem            `json:"folders"`
	Repos   []RepositorySummaryItem `json:"repos"`
}

type RepositorySummaryItem struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FolderID    *uint     `json:"folder_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateFolderInput struct {
	OwnerID  uint
	Name     string
	ParentID *uint
	Path     string // parent path; used when ParentID is nil
}

type RenameFolderInput struct {
	OwnerID uint
	ID      uint
	Name    string
}

type MoveFolderInput struct {
	OwnerID     uint
	ID          uint
	NewParentID *uint
}

type FolderUseCase struct {
	folders domain.FolderStore
	repos   domain.RepositoryStore
}

func NewFolderUseCase(folders domain.FolderStore, repos domain.RepositoryStore) *FolderUseCase {
	return &FolderUseCase{
		folders: folders,
		repos:   repos,
	}
}

func (uc *FolderUseCase) Create(in CreateFolderInput) (*FolderItem, error) {
	if in.OwnerID == 0 {
		return nil, ErrUnauthorized
	}

	name, slug, err := normalizeFolderName(in.Name)
	if err != nil {
		return nil, err
	}

	parentID := in.ParentID
	var parent *domain.Folder
	if parentID == nil && strings.TrimSpace(in.Path) != "" {
		parentPath := strings.Trim(strings.TrimSpace(in.Path), "/")
		parent, err = uc.folders.FindByOwnerAndPath(in.OwnerID, parentPath)
		if err != nil {
			return nil, ErrParentFolderNotFound
		}

		parentID = &parent.ID
	} else if parentID != nil {
		parent, err = uc.folders.FindByOwnerAndID(in.OwnerID, *parentID)
		if err != nil {
			return nil, ErrParentFolderNotFound
		}
	}

	depth := 1
	parentPath := ""
	if parent != nil {
		depth = folderDepth(parent.Path) + 1
		parentPath = parent.Path
	}

	if depth > maxFolderDepth {
		return nil, ErrFolderDepthExceeded
	}

	exists, err := uc.folders.ExistsByOwnerParentSlug(in.OwnerID, parentID, slug)
	if err != nil {
		return nil, fmt.Errorf("check folder existence: %w", err)
	}

	if exists {
		return nil, ErrFolderExists
	}

	path := slug
	if parentPath != "" {
		path = parentPath + "/" + slug
	}

	folder := &domain.Folder{
		OwnerID:  in.OwnerID,
		ParentID: parentID,
		Name:     name,
		Slug:     slug,
		Path:     path,
	}
	if err := uc.folders.Create(folder); err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}

	item := toFolderItem(folder)
	return &item, nil
}

func (uc *FolderUseCase) List(ownerID uint, parentID *uint, tree bool) (any, error) {
	if ownerID == 0 {
		return nil, ErrUnauthorized
	}

	if tree {
		all, err := uc.folders.ListByOwner(ownerID)
		if err != nil {
			return nil, fmt.Errorf("list folders: %w", err)
		}
		return buildFolderTree(all), nil
	}

	folders, err := uc.folders.ListByOwnerAndParent(ownerID, parentID)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}

	items := make([]FolderItem, 0, len(folders))
	for i := range folders {
		items = append(items, toFolderItem(&folders[i]))
	}

	return items, nil
}

func (uc *FolderUseCase) GetContents(ownerID, folderID uint) (*FolderContents, error) {
	if ownerID == 0 {
		return nil, ErrUnauthorized
	}

	folder, err := uc.folders.FindByOwnerAndID(ownerID, folderID)
	if err != nil {
		return nil, ErrFolderNotFound
	}

	children, err := uc.folders.ListByOwnerAndParent(ownerID, &folderID)
	if err != nil {
		return nil, fmt.Errorf("list child folders: %w", err)
	}

	repos, err := uc.repos.ListByOwnerAndFolderID(ownerID, &folderID)
	if err != nil {
		return nil, fmt.Errorf("list folder repos: %w", err)
	}

	folderItem := toFolderItem(folder)
	out := &FolderContents{
		Folder:  &folderItem,
		Folders: make([]FolderItem, 0, len(children)),
		Repos:   make([]RepositorySummaryItem, 0, len(repos)),
	}

	for i := range children {
		out.Folders = append(out.Folders, toFolderItem(&children[i]))
	}

	for _, repo := range repos {
		out.Repos = append(out.Repos, toRepoSummary(repo))
	}

	return out, nil
}

func (uc *FolderUseCase) Rename(in RenameFolderInput) (*FolderItem, error) {
	if in.OwnerID == 0 {
		return nil, ErrUnauthorized
	}

	folder, err := uc.folders.FindByOwnerAndID(in.OwnerID, in.ID)
	if err != nil {
		return nil, ErrFolderNotFound
	}

	name, slug, err := normalizeFolderName(in.Name)
	if err != nil {
		return nil, err
	}

	if slug != folder.Slug {
		exists, err := uc.folders.ExistsByOwnerParentSlug(in.OwnerID, folder.ParentID, slug)
		if err != nil {
			return nil, fmt.Errorf("check folder existence: %w", err)
		}
		if exists {
			return nil, ErrFolderExists
		}
	}

	oldPath := folder.Path
	newPath := slug
	if folder.ParentID != nil {
		parent, err := uc.folders.FindByOwnerAndID(in.OwnerID, *folder.ParentID)
		if err != nil {
			return nil, ErrParentFolderNotFound
		}
		newPath = parent.Path + "/" + slug
	}

	folder.Name = name
	folder.Slug = slug
	folder.Path = newPath
	if err := uc.folders.Update(folder); err != nil {
		return nil, fmt.Errorf("rename folder: %w", err)
	}

	if oldPath != newPath {
		if err := uc.rewriteDescendantPaths(in.OwnerID, oldPath, newPath); err != nil {
			return nil, err
		}
	}

	item := toFolderItem(folder)
	return &item, nil
}

func (uc *FolderUseCase) Delete(ownerID, folderID uint) error {
	if ownerID == 0 {
		return ErrUnauthorized
	}

	if _, err := uc.folders.FindByOwnerAndID(ownerID, folderID); err != nil {
		return ErrFolderNotFound
	}

	childFolders, err := uc.folders.CountChildren(folderID)
	if err != nil {
		return fmt.Errorf("count child folders: %w", err)
	}

	childRepos, err := uc.repos.CountByFolderID(folderID)
	if err != nil {
		return fmt.Errorf("count folder repos: %w", err)
	}

	if childFolders > 0 || childRepos > 0 {
		return ErrFolderNotEmpty
	}

	if err := uc.folders.Delete(folderID); err != nil {
		return fmt.Errorf("delete folder: %w", err)
	}

	return nil
}

func (uc *FolderUseCase) Move(in MoveFolderInput) (*FolderItem, error) {
	if in.OwnerID == 0 {
		return nil, ErrUnauthorized
	}
	if in.NewParentID != nil && *in.NewParentID == in.ID {
		return nil, ErrInvalidFolderMove
	}

	folder, err := uc.folders.FindByOwnerAndID(in.OwnerID, in.ID)
	if err != nil {
		return nil, ErrFolderNotFound
	}

	var newParent *domain.Folder
	if in.NewParentID != nil {
		newParent, err = uc.folders.FindByOwnerAndID(in.OwnerID, *in.NewParentID)
		if err != nil {
			return nil, ErrParentFolderNotFound
		}

		if newParent.Path == folder.Path || strings.HasPrefix(newParent.Path, folder.Path+"/") {
			return nil, ErrFolderCycle
		}
	}

	sameParent := (folder.ParentID == nil && in.NewParentID == nil) || (folder.ParentID != nil && in.NewParentID != nil && *folder.ParentID == *in.NewParentID)
	if sameParent {
		item := toFolderItem(folder)
		return &item, nil
	}

	depth := 1
	parentPath := ""
	if newParent != nil {
		depth = folderDepth(newParent.Path) + 1
		parentPath = newParent.Path
	}

	descendants, err := uc.folders.ListDescendants(in.OwnerID, folder.Path)
	if err != nil {
		return nil, fmt.Errorf("list descendants: %w", err)
	}

	maxRel := 0
	for _, d := range descendants {
		if d.ID == folder.ID {
			continue
		}

		rel := folderDepth(d.Path) - folderDepth(folder.Path)
		if rel > maxRel {
			maxRel = rel
		}
	}

	if depth+maxRel > maxFolderDepth {
		return nil, ErrFolderDepthExceeded
	}

	exists, err := uc.folders.ExistsByOwnerParentSlug(in.OwnerID, in.NewParentID, folder.Slug)
	if err != nil {
		return nil, fmt.Errorf("check folder existence: %w", err)
	}

	if exists {
		return nil, ErrFolderExists
	}

	oldPath := folder.Path
	newPath := folder.Slug
	if parentPath != "" {
		newPath = parentPath + "/" + folder.Slug
	}

	folder.ParentID = in.NewParentID
	folder.Path = newPath
	if err := uc.folders.Update(folder); err != nil {
		return nil, fmt.Errorf("move folder: %w", err)
	}

	if err := uc.rewriteDescendantPaths(in.OwnerID, oldPath, newPath); err != nil {
		return nil, err
	}

	item := toFolderItem(folder)
	return &item, nil
}

func (uc *FolderUseCase) rewriteDescendantPaths(ownerID uint, oldPath, newPath string) error {
	descendants, err := uc.folders.ListDescendants(ownerID, oldPath)
	if err != nil {
		return fmt.Errorf("list descendants: %w", err)
	}
	for i := range descendants {
		d := &descendants[i]
		if d.Path == oldPath {
			continue // already updated
		}

		if !strings.HasPrefix(d.Path, oldPath+"/") {
			continue
		}

		d.Path = newPath + d.Path[len(oldPath):]
		if err := uc.folders.Update(d); err != nil {
			return fmt.Errorf("update descendant path: %w", err)
		}
	}

	return nil
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

func folderDepth(path string) int {
	if path == "" {
		return 0
	}

	return strings.Count(path, "/") + 1
}

func toFolderItem(f *domain.Folder) FolderItem {
	return FolderItem{
		ID:        f.ID,
		ParentID:  f.ParentID,
		Name:      f.Name,
		Slug:      f.Slug,
		Path:      f.Path,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

func toRepoSummary(repo domain.Repository) RepositorySummaryItem {
	return RepositorySummaryItem{
		ID:          repo.ID,
		Name:        repo.Name,
		Description: repo.Description,
		FolderID:    repo.FolderID,
		CreatedAt:   repo.CreatedAt,
		UpdatedAt:   repo.UpdatedAt,
	}
}

func buildFolderTree(folders []domain.Folder) []FolderTreeNode {
	byParent := map[uint][]domain.Folder{}
	var roots []domain.Folder
	for _, f := range folders {
		if f.ParentID == nil {
			roots = append(roots, f)
			continue
		}

		byParent[*f.ParentID] = append(byParent[*f.ParentID], f)
	}

	var build func(f domain.Folder) FolderTreeNode
	build = func(f domain.Folder) FolderTreeNode {
		node := FolderTreeNode{
			FolderItem: toFolderItem(&f),
			Children:   []FolderTreeNode{},
		}

		for _, child := range byParent[f.ID] {
			node.Children = append(node.Children, build(child))
		}

		return node
	}

	tree := make([]FolderTreeNode, 0, len(roots))
	for _, root := range roots {
		tree = append(tree, build(root))
	}

	return tree
}
