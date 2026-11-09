package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v6/plumbing"

	"github.com/magomedcoder/repo/internal/config"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/ssrf"
	"github.com/magomedcoder/repo/pkg/syncutil"
	"github.com/magomedcoder/repo/pkg/urlredact"
)

var repoNameLocks syncutil.KeyedMutex

var (
	ErrNameRequired      = NewError(KindValidation, "repository_name_required")
	ErrAlreadyExists     = NewError(KindConflict, "repository_already_exists")
	ErrInvalidName       = NewError(KindValidation, "invalid_repository_name")
	ErrRepoNotFound      = NewError(KindNotFound, "repository_not_found")
	ErrRepoForbidden     = NewError(KindForbidden, "repository_access_denied")
	ErrRepoOwnerOnly     = NewError(KindForbidden, "repository_owner_only")
	ErrInvalidBranchName = NewError(KindValidation, "invalid_default_branch")
	ErrRepoQuota         = NewError(KindForbidden, "repository_quota_exceeded")
	ErrInvalidMirrorURL  = NewError(KindValidation, "invalid_mirror_url")
	ErrMirrorFailed      = NewError(KindUnavailable, "mirror_failed")
)

const maxReposPerOwner = 100

const nameHoldTTL = 30 * 24 * time.Hour

var repoNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}

type CreateRepositoryInput struct {
	Name            string
	Description     string
	ViewerID        uint
	OwnerID         uint
	Private         bool
	ShowOnHome      bool
	DefaultBranch   string
	MirrorDirection string
	MirrorURL       string
	MirrorUsername  string
	MirrorPassword  string

	MirrorSSHPrivateKey string
	MirrorSSHPassphrase string
	MirrorInterval      string
	MirrorRefspecs      string
	MirrorMaxBytes      int64
	MirrorGroupID       uint
}

type UpdateRepositoryInput struct {
	ViewerID             uint
	Name                 string
	NewName              *string
	Description          *string
	Private              *bool
	ShowOnHome           *bool
	DefaultBranch        *string
	ProtectDefaultBranch *bool
	IssuesEnabled        *bool
	CurrentPassword      string
}

type ResolveRepositoryInput struct {
	Name     string
	ViewerID uint
}

type RepositoryItem struct {
	ID                   uint       `json:"id"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	Owner                string     `json:"owner"`
	IsPrivate            bool       `json:"is_private"`
	ShowOnHome           bool       `json:"show_on_home"`
	ProtectDefaultBranch bool       `json:"protect_default_branch"`
	IssuesEnabled        bool       `json:"issues_enabled"`
	DefaultBranch        string     `json:"default_branch"`
	LastActivityAt       *time.Time `json:"last_activity_at,omitzero"`
	RepoPath             string     `json:"repo_path"`
	MirrorDirection      string     `json:"mirror_direction,omitzero"`
	MirrorURL            string     `json:"mirror_url,omitzero"`
	MirrorUsername       string     `json:"mirror_username,omitzero"`
	MirrorHasPassword    bool       `json:"mirror_has_password"`
	MirrorLastSyncAt     *time.Time `json:"mirror_last_sync_at,omitzero"`
	MirrorLastError      string     `json:"mirror_last_error,omitzero"`
	MirrorEnabled        *bool      `json:"mirror_enabled,omitzero"`
	MirrorStatus         string     `json:"mirror_status,omitzero"`
	MirrorReadOnly       bool       `json:"mirror_read_only,omitzero"`
	MirrorRemoteHEAD     string     `json:"mirror_remote_head,omitzero"`
	MirrorLastProgress   string     `json:"mirror_last_progress,omitzero"`
	MirrorInterval       string     `json:"mirror_interval,omitzero"`
	MirrorSyncWindow     string     `json:"mirror_sync_window,omitzero"`
	MirrorForce          *bool      `json:"mirror_force,omitzero"`
	MirrorPrune          *bool      `json:"mirror_prune,omitzero"`
	MirrorRefspecs       string     `json:"mirror_refspecs,omitzero"`
	MirrorHasSSHKey      bool       `json:"mirror_has_ssh_key,omitzero"`
	MirrorInsecureTLS    bool       `json:"mirror_insecure_tls,omitzero"`
	MirrorPinnedHosts    string     `json:"mirror_pinned_hosts,omitzero"`
	MirrorVerifySigned   bool       `json:"mirror_verify_signed,omitzero"`
	MirrorMaxBytes       int64      `json:"mirror_max_bytes,omitzero"`
	MirrorConflictPolicy string     `json:"mirror_conflict_policy,omitzero"`
	MirrorFailCount      int        `json:"mirror_fail_count,omitzero"`
	MirrorNextSyncAt     *time.Time `json:"mirror_next_sync_at,omitzero"`
	MirrorOwnerOnly      bool       `json:"mirror_owner_only,omitzero"`
	MirrorGroupID        uint       `json:"mirror_group_id,omitzero"`
	CanRead              bool       `json:"can_read"`
	CanWrite             bool       `json:"can_write"`
	CanAdmin             bool       `json:"can_admin"`
	CanOwn               bool       `json:"can_own"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type RepositoryUseCase struct {
	store            domain.RepositoryStore
	users            domain.UserStore
	git              domain.GitRepository
	activity         domain.ActivityStore
	authz            *Authz
	hasher           domain.PasswordHasher
	gate             *PasswordGate
	mirror           *mirrorRuntime
	mirrorSecretsKey []byte
	mirrorLookup     ssrf.LookupIPFunc
	codeIndex        domain.CodeIndexStore
	codeIdxRT        *codeIndexRuntime
	nameHolds        domain.RepoNameHoldStore
}

func NewRepositoryUseCase(
	store domain.RepositoryStore,
	users domain.UserStore,
	git domain.GitRepository,
	activity domain.ActivityStore,
	authz *Authz,
) *RepositoryUseCase {
	return &RepositoryUseCase{
		store:    store,
		users:    users,
		git:      git,
		activity: activity,
		authz:    authz,
		mirror:   newMirrorRuntime(),
	}
}

func (uc *RepositoryUseCase) SetNameHolds(h domain.RepoNameHoldStore) {
	if uc != nil {
		uc.nameHolds = h
	}
}

func (uc *RepositoryUseCase) holdName(ctx context.Context, name string) {
	if uc.nameHolds == nil || name == "" {
		return
	}

	if err := uc.nameHolds.Hold(ctx, name, time.Now().UTC().Add(nameHoldTTL)); err != nil {
		slog.Warn("name hold", "name", name, "err", err)
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

	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, err
	}

	unlock := repoNameLocks.Lock(strings.ToLower(name))
	defer unlock()

	ownerID := viewerID

	owner, err := uc.users.FindByID(ctx, ownerID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	ownerSlug := owner.Username

	if owner.Role != domain.RoleAdmin {
		n, cerr := uc.store.CountByOwnerID(ownerID)
		if cerr != nil {
			return nil, fmt.Errorf("count repositories: %w", cerr)
		}

		if n >= maxReposPerOwner {
			return nil, ErrRepoQuota
		}
	}

	if IsReservedSlug(name) {
		return nil, ErrReservedSlug
	}

	if err := uc.nameCollision(ctx, name); err != nil {
		return nil, err
	}

	mspec, err := uc.prepareCreateMirror(in)
	if err != nil {
		return nil, err
	}

	branch := strings.TrimSpace(in.DefaultBranch)
	if branch == "" {
		branch = config.DefaultBranch
	}

	if err := validateBranchName(branch); err != nil {
		return nil, err
	}

	logical := name + ".git"
	if _, err := domain.ParseRepoPath(logical); err != nil {
		return nil, ErrInvalidName
	}

	repoPath := buildRepoDiskPath(uc.git.GetReposRoot(), name)
	repo := &domain.Repository{
		Name:                 name,
		OwnerID:              ownerID,
		RepoPath:             logical,
		Description:          strings.TrimSpace(in.Description),
		IsPrivate:            in.Private,
		ShowOnHome:           !in.Private && in.ShowOnHome,
		ProtectDefaultBranch: true,
		IssuesEnabled:        true,
		DefaultBranch:        branch,
		Path:                 repoPath,
	}
	applyCreateMirror(repo, mspec)

	if err := uc.store.Create(repo); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}

		return nil, fmt.Errorf("save repository metadata: %w", err)
	}

	if mspec != nil && mspec.Direction == domain.MirrorPull {
		if err := uc.cloneNewMirror(ctx, repo, viewerID); err != nil {
			return nil, err
		}
	} else {
		if err := uc.git.InitBare(repoPath, domain.BareInitOptions{DefaultBranch: branch}); err != nil {
			_ = uc.store.Delete(repo.ID)
			_ = uc.git.Remove(repoPath)
			return nil, fmt.Errorf("init bare repository: %w", err)
		}
	}

	item := uc.toRepositoryItem(repo, ownerSlug, viewerID)
	recordActivity(uc.activity, &domain.ActivityEvent{
		ActorID: viewerID,
		OwnerID: ownerID,
		RepoID:  &repo.ID,
		Type:    domain.ActivityRepoCreate,
		Summary: activitySummary("created repository %s", name),
	})
	uc.ScheduleCodeIndex(repo.ID)

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
	repos, err := uc.store.ListByOwnerID(ctx, ownerID, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	return uc.mapRepos(repos, owner.Username)
}

func (uc *RepositoryUseCase) AnonymousReadDenied() bool {
	return uc != nil && uc.authz.AnonymousReadDenied()
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

	return uc.mapReposWithOwners(ctx, repos)
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

	return uc.mapReposWithOwners(ctx, repos)
}

func (uc *RepositoryUseCase) ListShared(ctx context.Context, userID uint, limit, offset int, order domain.RepoOrder) ([]RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if userID == 0 {
		return nil, ErrUnauthorized
	}

	offset, limit = NormalizePagination(offset, limit)
	repos, err := uc.store.ListSharedForUser(ctx, userID, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("list shared repositories: %w", err)
	}

	return uc.mapReposWithOwners(ctx, repos)
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
	repos, err := uc.store.SearchByName(ctx, q, ownerID, publicOnly || ownerID == 0, limit, offset, order)
	if err != nil {
		return nil, fmt.Errorf("search repositories: %w", err)
	}

	if publicOnly || ownerID == 0 {
		return uc.mapReposWithOwners(ctx, repos)
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

	repo, ownerSlug, err := uc.resolve(ctx, in)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	item := uc.toRepositoryItem(repo, ownerSlug, in.ViewerID)
	return &item, nil
}

func (uc *RepositoryUseCase) Update(ctx context.Context, in UpdateRepositoryInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, err := uc.resolve(ctx, ResolveRepositoryInput{
		Name:     in.Name,
		ViewerID: in.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	if updateNeedsOwner(repo, in) && !uc.authz.CanOwnRepo(repo, in.ViewerID) {
		return nil, ErrRepoOwnerOnly
	}

	if err := uc.requireUpdateStepUp(ctx, repo, in); err != nil {
		return nil, err
	}

	wasPrivate := repo.IsPrivate

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

	oldName := ""
	oldDisk := ""
	if in.NewName != nil {
		newName, nerr := normalizeRepoName(*in.NewName)
		if nerr != nil {
			return nil, nerr
		}

		if newName != repo.Name {
			unlock := repoNameLocks.LockAll(strings.ToLower(repo.Name), strings.ToLower(newName))
			defer unlock()

			if IsReservedSlug(newName) {
				return nil, ErrReservedSlug
			}

			if err := uc.nameCollision(ctx, newName); err != nil {
				return nil, err
			}

			oldName = repo.Name
			oldDisk = repo.Path
			repo.Name = newName
			repo.RepoPath = newName + ".git"
			repo.Path = buildRepoDiskPath(uc.git.GetReposRoot(), newName)
		}
	}

	if oldName != "" {
		if err := uc.git.Move(oldDisk, repo.Path); err != nil {
			return nil, fmt.Errorf("move repository: %w", err)
		}
	}

	if err := uc.store.Update(repo); err != nil {
		if oldName != "" {
			_ = uc.git.Move(repo.Path, oldDisk)
		}

		if isUniqueViolation(err) && oldName != "" {
			return nil, ErrAlreadyExists
		}

		return nil, fmt.Errorf("update repository: %w", err)
	}

	if oldName != "" {
		uc.holdName(ctx, oldName)
	}

	if wasPrivate && !repo.IsPrivate && uc.activity != nil {
		if err := uc.activity.DeleteByRepo(ctx, repo.ID); err != nil {
			slog.Warn("clear activity", "repo_id", repo.ID, "err", err)
		}
	}

	item := uc.toRepositoryItem(repo, ownerSlug, in.ViewerID)
	return &item, nil
}

func updateNeedsOwner(repo *domain.Repository, in UpdateRepositoryInput) bool {
	if in.Private != nil && *in.Private != repo.IsPrivate {
		return true
	}

	if in.NewName != nil {
		newName, err := normalizeRepoName(*in.NewName)
		return err != nil || newName != repo.Name
	}

	return false
}

func (uc *RepositoryUseCase) requireUpdateStepUp(ctx context.Context, repo *domain.Repository, in UpdateRepositoryInput) error {
	sensitive := in.Private != nil && *in.Private != repo.IsPrivate

	if !sensitive && in.NewName != nil {
		newName, err := normalizeRepoName(*in.NewName)
		if err != nil {
			return err
		}

		sensitive = newName != repo.Name
	}

	if !sensitive {
		return nil
	}

	return uc.requireViewerPassword(ctx, in.ViewerID, in.CurrentPassword)
}

type DeleteRepositoryInput struct {
	ResolveRepositoryInput
	Confirm         string
	CurrentPassword string
}

func (uc *RepositoryUseCase) Delete(ctx context.Context, in DeleteRepositoryInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if in.ViewerID == 0 {
		return ErrUnauthorized
	}

	repo, _, err := uc.resolve(ctx, in.ResolveRepositoryInput)
	if err != nil {
		return err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return ErrRepoNotFound
	}

	if !uc.authz.CanOwnRepo(repo, in.ViewerID) {
		return ErrRepoOwnerOnly
	}

	if strings.TrimSpace(in.Confirm) == "" {
		return ErrConfirmRequired
	}

	if in.Confirm != repo.Name {
		return ErrConfirmMismatch
	}

	if err := uc.requireViewerPassword(ctx, in.ViewerID, in.CurrentPassword); err != nil {
		return err
	}

	diskPath := repo.Path
	if err := uc.git.Remove(diskPath); err != nil {
		return fmt.Errorf("delete repository files: %w", err)
	}

	if err := uc.store.Delete(repo.ID); err != nil {
		return fmt.Errorf("delete repository metadata: %w", err)
	}

	uc.holdName(ctx, repo.Name)
	uc.releaseMirrorRuntime(repo.ID)
	uc.DropCodeIndex(ctx, repo.ID)
	if uc.mirror.events != nil {
		if err := uc.mirror.events.DeleteByRepo(ctx, repo.ID); err != nil {
			slog.Warn("mirror events", "repo_id", repo.ID, "err", err)
		}
	}

	return nil
}

func (uc *RepositoryUseCase) SetPasswordHasher(h domain.PasswordHasher) {
	if uc != nil {
		uc.hasher = h
		uc.gate = NewPasswordGate(h)
	}
}

func (uc *RepositoryUseCase) SetPasswordGate(g *PasswordGate) {
	if uc != nil && g != nil {
		uc.gate = g
	}
}

func (uc *RepositoryUseCase) requireViewerPassword(ctx context.Context, viewerID uint, password string) error {
	if uc == nil || uc.gate == nil || uc.users == nil {
		return ErrPasswordRequired
	}

	password = strings.TrimSpace(password)
	if password == "" {
		return ErrPasswordRequired
	}

	user, err := uc.users.FindByID(ctx, viewerID)
	if err != nil {
		return ErrUnauthorized
	}

	return uc.gate.Verify(user.ID, user.PasswordHash, password, ErrCurrentPasswordWrong)
}

func (uc *RepositoryUseCase) PurgeSoftDeletedRepos(ctx context.Context, olderThan time.Duration) (int, error) {
	paths, ids, err := uc.store.PurgeSoftDeleted(ctx, time.Now().UTC().Add(-olderThan))
	if err != nil {
		return 0, fmt.Errorf("purge soft-deleted repositories: %w", err)
	}

	var errs []error
	for _, path := range paths {
		if err := uc.git.Remove(path); err != nil {
			errs = append(errs, fmt.Errorf("remove purged repository %q: %w", path, err))
		}
	}

	for _, id := range ids {
		uc.releaseMirrorRuntime(id)
		uc.DropCodeIndex(ctx, id)
		if uc.mirror.events != nil {
			if err := uc.mirror.events.DeleteByRepo(ctx, id); err != nil {
				errs = append(errs, fmt.Errorf("delete mirror events for repository %d: %w", id, err))
			}
		}
	}

	return len(ids), errors.Join(errs...)
}

func (uc *RepositoryUseCase) resolve(ctx context.Context, in ResolveRepositoryInput) (*domain.Repository, string, error) {
	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, "", err
	}

	logical := name + ".git"
	if _, err := domain.ParseRepoPath(logical); err != nil {
		return nil, "", ErrRepoNotFound
	}

	repo, err := uc.store.FindByRepoPath(ctx, logical)
	if err != nil {
		if in.ViewerID == 0 && uc.authz.AnonymousReadDenied() {
			return nil, "", ErrUnauthorized
		}

		return nil, "", ErrRepoNotFound
	}

	ownerSlug := ""
	if user, uerr := uc.users.FindByID(ctx, repo.OwnerID); uerr == nil {
		ownerSlug = user.Username
	}

	return repo, ownerSlug, nil
}

func (uc *RepositoryUseCase) mapRepos(repos []domain.Repository, ownerUsername string) ([]RepositoryItem, error) {
	items := make([]RepositoryItem, 0, len(repos))
	for _, repo := range repos {
		items = append(items, uc.toRepositoryItem(&repo, ownerUsername, repo.OwnerID))
	}

	return items, nil
}

func (uc *RepositoryUseCase) mapReposWithOwners(ctx context.Context, repos []domain.Repository) ([]RepositoryItem, error) {
	items := make([]RepositoryItem, 0, len(repos))
	userCache := map[uint]string{}
	for _, repo := range repos {
		username, ok := userCache[repo.OwnerID]
		if !ok {
			user, err := uc.users.FindByID(ctx, repo.OwnerID)
			if err != nil {
				continue
			}

			username = user.Username
			userCache[repo.OwnerID] = username
		}

		items = append(items, uc.toRepositoryItem(&repo, username, 0))
	}

	return items, nil
}

func buildRepoDiskPath(basePath, name string) string {
	return filepath.Join(basePath, name+".git")
}

func normalizeRepoName(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return "", ErrNameRequired
	}

	if strings.HasSuffix(name, ".git") {
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

func (uc *RepositoryUseCase) nameCollision(ctx context.Context, name string) error {
	exists, err := uc.store.ExistsByName(name)
	if err != nil {
		return fmt.Errorf("check repository existence: %w", err)
	}

	if exists {
		return ErrAlreadyExists
	}

	if uc.nameHolds != nil {
		held, herr := uc.nameHolds.IsHeld(ctx, name, time.Now().UTC())
		if herr != nil {
			return fmt.Errorf("check repository name hold: %w", herr)
		}

		if held {
			return ErrAlreadyExists
		}
	}

	return nil
}

func (uc *RepositoryUseCase) toRepositoryItem(repo *domain.Repository, ownerUsername string, viewerID uint) RepositoryItem {
	item := RepositoryItem{
		ID:                   repo.ID,
		Name:                 repo.Name,
		Description:          repo.Description,
		Owner:                ownerUsername,
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
		item.CanOwn = uc.authz.CanOwnRepo(repo, viewerID)
	} else if !repo.IsPrivate {
		item.CanRead = true
	}

	if repo.MirrorDirection != domain.MirrorNone {
		fillMirrorItem(&item, repo)
	}

	return item
}

func fillMirrorItem(item *RepositoryItem, repo *domain.Repository) {
	enabled := repo.MirrorEnabled
	item.MirrorDirection = repo.MirrorDirection
	item.MirrorEnabled = &enabled
	item.MirrorStatus = effectiveMirrorStatus(repo)
	item.MirrorReadOnly = repo.MirrorDirection == domain.MirrorPull

	if !item.CanAdmin {
		return
	}

	force, prune := repo.MirrorForce, repo.MirrorPrune
	item.MirrorURL = urlredact.URL(repo.MirrorURL)
	item.MirrorLastSyncAt = repo.MirrorLastSyncAt
	item.MirrorRemoteHEAD = repo.MirrorRemoteHEAD
	item.MirrorLastProgress = repo.MirrorLastProgress
	item.MirrorUsername = repo.MirrorUsername
	item.MirrorHasPassword = repo.MirrorPassword != ""
	item.MirrorLastError = repo.MirrorLastError
	item.MirrorInterval = repo.MirrorInterval
	item.MirrorSyncWindow = repo.MirrorSyncWindow
	item.MirrorForce, item.MirrorPrune = &force, &prune
	item.MirrorRefspecs = repo.MirrorRefspecs
	item.MirrorHasSSHKey = repo.MirrorSSHPrivateKey != ""
	item.MirrorInsecureTLS = repo.MirrorInsecureTLS
	item.MirrorPinnedHosts = repo.MirrorPinnedHosts
	item.MirrorVerifySigned = repo.MirrorVerifySigned
	item.MirrorMaxBytes = repo.MirrorMaxBytes
	item.MirrorConflictPolicy = repo.MirrorConflictPolicy
	item.MirrorFailCount = repo.MirrorFailCount
	item.MirrorNextSyncAt = repo.MirrorNextSyncAt
	item.MirrorOwnerOnly = repo.MirrorOwnerOnly
	item.MirrorGroupID = repo.MirrorGroupID
}
