package usecase

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/urlredact"
)

var (
	ErrMirrorGroupNotFound = NewError(KindNotFound, "mirror_group_not_found")
	ErrMirrorImportLimit   = NewError(KindValidation, "mirror_import_limit")
)

const (
	maxImportMirrors     = 50
	maxAdminMirrorList   = 500
	maxAdminMirrorVerify = 100
	maxBulkSyncMirrors   = 200
)

type AdminMirrorItem struct {
	ID         uint       `json:"id"`
	RepoID     uint       `json:"repo_id"`
	RepoPath   string     `json:"repo_path"`
	Owner      string     `json:"owner"`
	Direction  string     `json:"direction"`
	URL        string     `json:"url"`
	Enabled    bool       `json:"enabled"`
	Paused     bool       `json:"paused"`
	Status     string     `json:"status"`
	LastSyncAt *time.Time `json:"last_sync_at,omitzero"`
	LastError  string     `json:"last_error,omitzero"`
	NextSyncAt *time.Time `json:"next_sync_at,omitzero"`
	FailCount  int        `json:"fail_count"`
	GroupID    uint       `json:"group_id,omitzero"`
}

func (uc *RepositoryUseCase) ListAdminMirrors(ctx context.Context, viewerID uint, filter domain.MirrorListFilter) ([]AdminMirrorItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return nil, err
	}

	if filter.Limit <= 0 || filter.Limit > maxAdminMirrorList {
		filter.Limit = 100
	}

	repos, err := uc.store.ListMirrors(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list mirrors: %w", err)
	}

	owners := map[uint]string{}
	out := make([]AdminMirrorItem, 0, len(repos))

	for i := range repos {
		r := &repos[i]

		owner, ok := owners[r.OwnerID]
		if !ok {
			if u, uerr := uc.users.FindByID(ctx, r.OwnerID); uerr == nil {
				owner = u.Username
			}

			owners[r.OwnerID] = owner
		}

		out = append(out, AdminMirrorItem{
			ID:         r.ID,
			RepoID:     r.ID,
			RepoPath:   r.RepoPath,
			Owner:      owner,
			Direction:  r.MirrorDirection,
			URL:        urlredact.URL(r.MirrorURL),
			Enabled:    r.MirrorEnabled,
			Paused:     !r.MirrorEnabled,
			Status:     effectiveMirrorStatus(r),
			LastSyncAt: r.MirrorLastSyncAt,
			LastError:  r.MirrorLastError,
			NextSyncAt: r.MirrorNextSyncAt,
			FailCount:  r.MirrorFailCount,
			GroupID:    r.MirrorGroupID,
		})
	}

	return out, nil
}

func (uc *RepositoryUseCase) BulkSyncAllMirrors(ctx context.Context, viewerID uint) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return 0, err
	}

	repos, err := uc.listAllMirrors(ctx, domain.MirrorListFilter{})
	if err != nil {
		return 0, err
	}

	ids := make([]uint, 0, len(repos))
	for i := range repos {
		if repos[i].MirrorEnabled && repos[i].MirrorDirection != domain.MirrorNone {
			ids = append(ids, repos[i].ID)
			if len(ids) >= maxBulkSyncMirrors {
				break
			}
		}
	}

	uc.goMirror(func() {
		uc.syncBatch(uc.mirrorRootContext(), ids, domain.MirrorTriggerBulk, viewerID)
	})

	return len(ids), nil
}

func (uc *RepositoryUseCase) BulkPauseMirrors(ctx context.Context, viewerID uint, repoIDs []uint, pause bool) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return 0, err
	}

	changed := 0

	for _, id := range repoIDs {
		repo, err := uc.store.FindByID(ctx, id)
		if err != nil || repo == nil || repo.MirrorDirection == domain.MirrorNone || repo.MirrorEnabled == !pause {
			continue
		}

		prev := *repo
		repo.MirrorEnabled = !pause
		uc.settleMirrorStatus(repo, &prev)

		if err := uc.store.Update(repo); err != nil {
			continue
		}

		uc.auditConfigure(repo, &prev, viewerID)

		changed++
	}

	return changed, nil
}

func (uc *RepositoryUseCase) BulkSyncMirrors(ctx context.Context, viewerID uint, repoIDs []uint) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return 0, err
	}

	ids := make([]uint, 0, len(repoIDs))

	for _, id := range repoIDs {
		repo, err := uc.store.FindByID(ctx, id)
		if err != nil || repo == nil || repo.MirrorDirection == domain.MirrorNone || !repo.MirrorEnabled {
			continue
		}

		ids = append(ids, id)
		if len(ids) >= maxBulkSyncMirrors {
			break
		}
	}

	uc.goMirror(func() {
		uc.syncBatch(uc.mirrorRootContext(), ids, domain.MirrorTriggerBulk, viewerID)
	})

	return len(ids), nil
}

type MirrorHealthItem struct {
	InError   int64  `json:"in_error"`
	OlderThan string `json:"older_than"`
	Healthy   bool   `json:"healthy"`
}

func (uc *RepositoryUseCase) MirrorHealth(ctx context.Context, olderThan time.Duration) (*MirrorHealthItem, error) {
	if olderThan <= 0 {
		olderThan = time.Hour
	}

	n, err := uc.store.CountMirrorsInError(ctx, time.Now().Add(-olderThan))
	if err != nil {
		return nil, fmt.Errorf("mirror health: %w", err)
	}

	return &MirrorHealthItem{
		InError:   n,
		OlderThan: olderThan.String(),
		Healthy:   n == 0,
	}, nil
}

func (uc *RepositoryUseCase) CreateMirrorGroup(ctx context.Context, viewerID uint, g domain.MirrorGroup) (*domain.MirrorGroup, error) {
	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return nil, err
	}

	if uc.mirror.groups == nil {
		return nil, ErrMirrorNotConfigured
	}

	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" {
		return nil, ErrNameRequired
	}

	if g.UpstreamURL != "" {
		if err := uc.validateMirrorURL(g.UpstreamURL); err != nil {
			return nil, err
		}
	}

	iv, err := normalizeInterval(g.Interval)
	if err != nil {
		return nil, err
	}

	g.Interval = iv

	dir, err := normalizeMirrorDirection(g.Direction)
	if err != nil {
		return nil, err
	}

	if dir == domain.MirrorNone {
		dir = domain.MirrorPull
	}

	g.Direction = dir

	if err := uc.mirror.groups.Create(ctx, &g); err != nil {
		return nil, fmt.Errorf("create mirror group: %w", err)
	}

	return &g, nil
}

func (uc *RepositoryUseCase) ListMirrorGroups(ctx context.Context, viewerID uint) ([]domain.MirrorGroup, error) {
	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return nil, err
	}

	if uc.mirror.groups == nil {
		return nil, nil
	}

	return uc.mirror.groups.List(ctx)
}

func (uc *RepositoryUseCase) DeleteMirrorGroup(ctx context.Context, viewerID, groupID uint) error {
	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return err
	}

	if uc.mirror.groups == nil {
		return ErrMirrorGroupNotFound
	}

	if _, err := uc.mirror.groups.FindByID(ctx, groupID); err != nil {
		return ErrMirrorGroupNotFound
	}

	return uc.mirror.groups.Delete(ctx, groupID)
}

type ImportMirrorsInput struct {
	ViewerID uint
	URLs     []string
	Username string
	Password string
	Private  bool
	GroupID  uint
	Interval string
}

type ImportMirrorResult struct {
	URL     string `json:"url"`
	Name    string `json:"name,omitzero"`
	Created bool   `json:"created"`
	Error   string `json:"error,omitzero"`
}

func repoNameFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	p := raw

	if u, err := url.Parse(raw); err == nil && u.Path != "" && strings.Contains(raw, "://") {
		p = u.Path
	} else if i := strings.LastIndex(raw, ":"); i >= 0 {
		p = raw[i+1:]
	}

	name := path.Base(strings.TrimSuffix(strings.TrimRight(p, "/"), ".git"))
	if name == "." || name == "/" {
		return ""
	}

	return name
}

func (uc *RepositoryUseCase) ImportOrgMirrors(ctx context.Context, in ImportMirrorsInput) ([]ImportMirrorResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	if len(in.URLs) == 0 || len(in.URLs) > maxImportMirrors {
		return nil, ErrMirrorImportLimit
	}

	interval, groupID, err := uc.importGroupDefaults(ctx, in)
	if err != nil {
		return nil, err
	}

	results := make([]ImportMirrorResult, 0, len(in.URLs))

	for _, raw := range in.URLs {
		raw = strings.TrimSpace(raw)
		res := ImportMirrorResult{
			URL:  urlredact.URL(raw),
			Name: repoNameFromURL(raw),
		}

		_, cerr := uc.Create(ctx, CreateRepositoryInput{
			Name:            res.Name,
			ViewerID:        in.ViewerID,
			OwnerID:         in.ViewerID,
			Private:         in.Private,
			MirrorDirection: domain.MirrorPull,
			MirrorURL:       raw,
			MirrorUsername:  in.Username,
			MirrorPassword:  in.Password,
			MirrorInterval:  interval,
			MirrorGroupID:   groupID,
		})

		if cerr != nil {
			res.Error = cerr.Error()
		} else {
			res.Created = true
		}

		results = append(results, res)
	}

	return results, nil
}

func (uc *RepositoryUseCase) importGroupDefaults(ctx context.Context, in ImportMirrorsInput) (string, uint, error) {
	interval, err := normalizeInterval(in.Interval)
	if err != nil {
		return "", 0, err
	}

	if in.GroupID == 0 {
		return interval, 0, nil
	}

	if uc.mirror.groups == nil {
		return "", 0, ErrMirrorGroupNotFound
	}

	g, err := uc.mirror.groups.FindByID(ctx, in.GroupID)
	if err != nil {
		return "", 0, ErrMirrorGroupNotFound
	}

	if interval == "" {
		interval = g.Interval
	}

	return interval, g.ID, nil
}
