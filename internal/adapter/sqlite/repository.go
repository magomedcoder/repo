package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

func NewDB(dataDir string) (*gorm.DB, error) {
	dsn := filepath.Join(dataDir, "db.sqlite")
	file, _, _ := strings.Cut(dsn, "?")

	if dir := filepath.Dir(file); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("data dir: %w", err)
		}
	}

	if !strings.Contains(dsn, "_pragma") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := runMigrations(db, file); err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}

		return nil, fmt.Errorf("migrate: %w", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	if file != "" && file != ":memory:" {
		_ = os.Chmod(file, 0o600)
	}

	return db, nil
}

type Repository struct {
	gormutil.Repo[model.Repository]
}

func NewRepositoryRepo(db *gorm.DB) *Repository {
	return &Repository{Repo: gormutil.NewRepo[model.Repository](db)}
}

func (s *Repository) Create(repo *domain.Repository) error {
	m := model.RepositoryFromDomain(repo)
	if err := s.Repo.Create(context.Background(), m); err != nil {
		return err
	}

	*repo = m.ToDomain()
	return nil
}

func (s *Repository) Update(repo *domain.Repository) error {
	m := model.RepositoryFromDomain(repo)
	if err := s.Save(context.Background(), m, gormutil.WithOmit("IssueSeq", "PullSeq")); err != nil {
		return err
	}

	*repo = m.ToDomain()
	return nil
}

func (s *Repository) Delete(id uint) error {
	return s.DeleteByID(context.Background(), id)
}

func (s *Repository) PurgeSoftDeleted(ctx context.Context, before time.Time) ([]uint, error) {
	var ids []uint
	err := s.Txx(ctx, func(tx *gorm.DB) error {
		if err := tx.Unscoped().Model(&model.Repository{}).
			Where("deleted_at IS NOT NULL AND deleted_at < ?", before.In(time.Local)).
			Pluck("id", &ids).Error; err != nil {
			return err
		}

		if len(ids) == 0 {
			return nil
		}

		stmts := []string{
			`DELETE FROM issue_labels WHERE issue_id IN (SELECT id FROM issues WHERE repository_id IN ?)`,
			`DELETE FROM issue_comments WHERE issue_id IN (SELECT id FROM issues WHERE repository_id IN ?)`,
			`DELETE FROM issues WHERE repository_id IN ?`,
			`DELETE FROM pull_request_comments WHERE pull_id IN (SELECT id FROM pull_requests WHERE repository_id IN ?)`,
			`DELETE FROM pull_requests WHERE repository_id IN ?`,
			`DELETE FROM labels WHERE repository_id IN ?`,
			`DELETE FROM team_repo_grants WHERE repo_id IN ?`,
			`DELETE FROM repo_collaborators WHERE repo_id IN ?`,
			`DELETE FROM mirror_sync_events WHERE repo_id IN ?`,
			`DELETE FROM activity_events WHERE repo_id IN ?`,
			`DELETE FROM repositories WHERE id IN ?`,
		}
		for _, stmt := range stmts {
			if err := tx.Exec(stmt, ids).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *Repository) FindByRepoPath(ctx context.Context, repoPath string) (*domain.Repository, error) {
	row, err := s.FindByWhere(ctx, "repo_path = ?", repoPath)
	if err != nil {
		return nil, mapDBErr(err)
	}

	repo := row.ToDomain()
	return &repo, nil
}

func (s *Repository) ListByOwnerID(ctx context.Context, ownerID uint, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	limit, offset = clampListBounds(limit, offset)
	models, err := s.FindAll(ctx,
		gormutil.WithWhere("owner_id = ?", ownerID),
		gormutil.WithOrder(repoOrderClause(order)),
		gormutil.LimitOffset(limit, offset),
	)
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *Repository) ListPublic(ctx context.Context, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	limit, offset = clampListBounds(limit, offset)
	models, err := s.FindAll(ctx,
		gormutil.WithWhere("is_private = ?", false),
		gormutil.WithOrder(repoOrderClause(order)),
		gormutil.LimitOffset(limit, offset),
	)
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *Repository) ListHome(ctx context.Context, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	limit, offset = clampListBounds(limit, offset)
	models, err := s.FindAll(ctx,
		gormutil.WithWhere("is_private = ? AND show_on_home = ?", false, true),
		gormutil.WithOrder(repoOrderClause(order)),
		gormutil.LimitOffset(limit, offset),
	)
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *Repository) ListSharedForUser(ctx context.Context, userID uint, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	if userID == 0 {
		return nil, nil
	}

	limit, offset = clampListBounds(limit, offset)
	var models []model.Repository
	err := s.Model(ctx).
		Where("owner_id != ?", userID).
		Where(`(
			id IN (
				SELECT repo_id FROM repo_collaborators
				WHERE user_id = ? AND deleted_at IS NULL
			)
			OR id IN (
				SELECT team_repo_grants.repo_id
				FROM team_repo_grants
				JOIN team_members ON team_members.team_id = team_repo_grants.team_id
					AND team_members.deleted_at IS NULL
				WHERE team_members.user_id = ?
					AND team_repo_grants.deleted_at IS NULL
			)
		)`, userID, userID).
		Order(repoOrderClause(order)).
		Limit(limit).
		Offset(offset).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *Repository) SearchByName(ctx context.Context, query string, ownerID uint, publicOnly bool, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	q := "%" + escapeLike(query) + "%"
	db := s.Model(ctx).Where(`(name LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\')`, q, q)
	if publicOnly {
		db = db.Where("is_private = ?", false)
	} else if ownerID > 0 {
		db = db.Where("owner_id = ?", ownerID)
	} else {
		db = db.Where("is_private = ?", false)
	}

	limit, offset = clampListBounds(limit, offset)
	var models []model.Repository
	if err := db.Order(repoOrderClause(order)).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func accessibleRepoScope(db *gorm.DB, viewerID uint) *gorm.DB {
	if viewerID == 0 {
		return db.Where("is_private = ?", false)
	}

	return db.Where(`(
		is_private = ?
		OR owner_id = ?
		OR id IN (
			SELECT repo_id FROM repo_collaborators
			WHERE user_id = ? AND deleted_at IS NULL
		)
		OR id IN (
			SELECT team_repo_grants.repo_id
			FROM team_repo_grants
			JOIN team_members ON team_members.team_id = team_repo_grants.team_id
				AND team_members.deleted_at IS NULL
			WHERE team_members.user_id = ?
				AND team_repo_grants.deleted_at IS NULL
		)
	)`, false, viewerID, viewerID, viewerID)
}

func (s *Repository) SearchAccessible(ctx context.Context, query string, viewerID uint, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	q := "%" + escapeLike(query) + "%"
	limit, offset = clampListBounds(limit, offset)
	db := accessibleRepoScope(s.Model(ctx), viewerID).
		Where(`(name LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\')`, q, q)

	var models []model.Repository
	if err := db.Order(repoOrderClause(order)).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *Repository) ListAccessible(ctx context.Context, viewerID uint, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	limit, offset = clampListBounds(limit, offset)
	db := accessibleRepoScope(s.Model(ctx), viewerID)

	var models []model.Repository
	if err := db.Order(repoOrderClause(order)).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *Repository) ExistsByName(name string) (bool, error) {
	return s.QueryExistOpts(context.Background(), "", nil,
		gormutil.WithUnscoped(),
		gormutil.WithWhere("name = ?", name),
	)
}

func (s *Repository) TouchLastActivity(id uint, at time.Time) error {
	_, err := s.UpdateByID(context.Background(), id, map[string]any{
		"last_activity_at": at,
		"updated_at":       at,
	})
	return err
}

func (s *Repository) FindByID(ctx context.Context, id uint) (*domain.Repository, error) {
	row, err := s.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	repo := row.ToDomain()

	return &repo, nil
}

func (s *Repository) ListMirrors(ctx context.Context, f domain.MirrorListFilter) ([]domain.Repository, error) {
	q := s.Model(ctx).Where("mirror_direction != '' AND mirror_url != ''")
	if f.Direction != "" {
		q = q.Where("mirror_direction = ?", f.Direction)
	}

	if f.Status != "" {
		q = q.Where("mirror_status = ?", f.Status)
	}

	if f.GroupID != 0 {
		q = q.Where("mirror_group_id = ?", f.GroupID)
	}

	if f.OnlyError {
		q = q.Where("(mirror_status = ? OR mirror_last_error != '')", domain.MirrorStatusError)
	}

	if text := strings.TrimSpace(f.Query); text != "" {
		like := "%" + escapeLike(text) + "%"
		q = q.Where(`(name LIKE ? ESCAPE '\' OR repo_path LIKE ? ESCAPE '\' OR mirror_url LIKE ? ESCAPE '\')`, like, like, like)
	}

	limit, offset := f.Limit, f.Offset
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	var models []model.Repository
	if err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *Repository) ListMirrorSchedules(ctx context.Context, f domain.MirrorScheduleFilter) ([]domain.MirrorSchedule, error) {
	limit, offset := f.Limit, f.Offset
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	type row struct {
		ID               uint
		MirrorDirection  string
		MirrorURL        string
		MirrorEnabled    bool
		MirrorInterval   string
		MirrorSyncWindow string
		MirrorNextSyncAt *time.Time
		MirrorLastSyncAt *time.Time
	}

	q := s.Model(ctx).
		Select("id, mirror_direction, mirror_url, mirror_enabled, mirror_interval, mirror_sync_window, mirror_next_sync_at, mirror_last_sync_at").
		Where("mirror_direction != '' AND mirror_url != ''")

	if f.DueBefore != nil {
		q = q.Where("mirror_enabled = ? AND (mirror_next_sync_at IS NULL OR mirror_next_sync_at <= ?)", true, *f.DueBefore)
	}

	var rows []row
	err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.MirrorSchedule, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.MirrorSchedule{
			ID:         r.ID,
			Direction:  r.MirrorDirection,
			URL:        r.MirrorURL,
			Enabled:    r.MirrorEnabled,
			Interval:   r.MirrorInterval,
			SyncWindow: r.MirrorSyncWindow,
			NextSyncAt: r.MirrorNextSyncAt,
			LastSyncAt: r.MirrorLastSyncAt,
		})
	}

	return out, nil
}

func (s *Repository) ResetStaleMirrorSyncing(ctx context.Context) (int64, error) {
	return s.UpdateColumnWhere(ctx, "mirror_status", domain.MirrorStatusIdle, "mirror_status = ?", domain.MirrorStatusSyncing)
}

func (s *Repository) CountMirrorsInError(ctx context.Context, since time.Time) (int64, error) {
	return s.QueryCountOpts(ctx,
		"mirror_direction != '' AND mirror_url != '' AND mirror_enabled = ? AND mirror_status = ? AND (mirror_last_sync_at IS NULL OR mirror_last_sync_at < ?)",
		[]any{true, domain.MirrorStatusError, since},
	)
}

func (s *Repository) CountAll(ctx context.Context) (int64, error) {
	return s.QueryCount(ctx, "1 = 1")
}

func (s *Repository) ListAdmin(ctx context.Context, filter domain.AdminRepoFilter) ([]domain.Repository, int, error) {
	q := s.Model(ctx)
	if text := strings.TrimSpace(filter.Query); text != "" {
		like := "%" + escapeLike(text) + "%"
		q = q.Where(`(name LIKE ? ESCAPE '\' OR repo_path LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\')`, like, like, like)
	}

	if filter.Private != nil {
		q = q.Where("is_private = ?", *filter.Private)
	}

	if filter.Mirror {
		q = q.Where("mirror_direction != '' AND mirror_url != ''")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit, offset := clampListBounds(filter.Limit, filter.Offset)
	var models []model.Repository
	if err := q.Order(repoOrderClause(filter.Order)).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	return repoModelsToDomain(models), int(total), nil
}

func (s *Repository) UpdateMirrorState(ctx context.Context, repo *domain.Repository) error {
	_, err := s.UpdateByID(ctx, repo.ID, map[string]any{
		"default_branch":       repo.DefaultBranch,
		"mirror_last_sync_at":  repo.MirrorLastSyncAt,
		"mirror_last_error":    repo.MirrorLastError,
		"mirror_status":        repo.MirrorStatus,
		"mirror_fail_count":    repo.MirrorFailCount,
		"mirror_next_sync_at":  repo.MirrorNextSyncAt,
		"mirror_last_progress": repo.MirrorLastProgress,
		"mirror_remote_head":   repo.MirrorRemoteHEAD,
	})
	return err
}

func repoOrderClause(order domain.RepoOrder) string {
	if order == domain.RepoOrderName {
		return "name COLLATE NOCASE ASC, id ASC"
	}

	return "updated_at DESC"
}

func (s *Repository) CountByOwnerID(ownerID uint) (int64, error) {
	return s.QueryCount(context.Background(), "owner_id = ?", ownerID)
}

func clampListBounds(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	if offset < 0 {
		offset = 0
	}

	if offset > 10000 {
		offset = 10000
	}

	return limit, offset
}

func (s *Repository) ListPublicIDsByOwner(ctx context.Context, ownerID uint) ([]uint, error) {
	var ids []uint
	err := s.Pluck(ctx, "id", &ids, "owner_id = ? AND is_private = ?", ownerID, false)
	return ids, err
}

func repoModelsToDomain(models []model.Repository) []domain.Repository {
	repos := make([]domain.Repository, 0, len(models))
	for _, m := range models {
		repos = append(repos, m.ToDomain())
	}

	return repos
}
