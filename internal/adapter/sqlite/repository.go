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
	"magomedcoder.ru/repo/internal/domain"
)

type RepositoryStore struct {
	db *gorm.DB
}

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

	if file != "" && file != ":memory:" {
		_ = os.Chmod(file, 0o600)
	}

	return db, nil
}

func NewRepositoryStore(db *gorm.DB) *RepositoryStore {
	return &RepositoryStore{db: db}
}

func (s *RepositoryStore) Create(repo *domain.Repository) error {
	model := fromDomain(repo)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*repo = model.toDomain()
	return nil
}

func (s *RepositoryStore) Update(repo *domain.Repository) error {
	model := fromDomain(repo)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*repo = model.toDomain()
	return nil
}

func (s *RepositoryStore) Delete(id uint) error {
	return s.db.Delete(&repositoryModel{}, id).Error
}

func (s *RepositoryStore) FindByRepoPath(ctx context.Context, repoPath string) (*domain.Repository, error) {
	var model repositoryModel
	err := s.db.WithContext(ctx).Where("repo_path = ?", repoPath).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}

	repo := model.toDomain()
	return &repo, nil
}

func (s *RepositoryStore) ListByOwnerID(ctx context.Context, ownerKind string, ownerID uint, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	var models []repositoryModel
	limit, offset = clampListBounds(limit, offset)
	q := s.db.WithContext(ctx).Where("owner_kind = ? AND owner_id = ?", ownerKind, ownerID).Order(repoOrderClause(order)).Limit(limit).Offset(offset)

	err := q.Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ListByOwnerAndFolderID(ctx context.Context, ownerKind string, ownerID uint, folderID *uint) ([]domain.Repository, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}
	var models []repositoryModel
	q := s.db.WithContext(ctx).Where("owner_kind = ? AND owner_id = ?", ownerKind, ownerID)
	if folderID == nil {
		q = q.Where("folder_id IS NULL")
	} else {
		q = q.Where("folder_id = ?", *folderID)
	}

	err := q.Order("name ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ListByFolderID(ctx context.Context, folderID uint) ([]domain.Repository, error) {
	var models []repositoryModel
	err := s.db.WithContext(ctx).Where("folder_id = ?", folderID).Order("name ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ListPublic(ctx context.Context, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	var models []repositoryModel
	limit, offset = clampListBounds(limit, offset)
	err := s.db.WithContext(ctx).Where("is_private = ?", false).Order(repoOrderClause(order)).Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ListHome(ctx context.Context, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	var models []repositoryModel
	limit, offset = clampListBounds(limit, offset)
	err := s.db.WithContext(ctx).Where("is_private = ? AND show_on_home = ?", false, true).Order(repoOrderClause(order)).Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) SearchByName(ctx context.Context, query string, ownerKind string, ownerID uint, publicOnly bool, limit, offset int, order domain.RepoOrder) ([]domain.Repository, error) {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
	q := "%" + escaped + "%"
	db := s.db.WithContext(ctx).Model(&repositoryModel{}).Where(`name LIKE ? ESCAPE '\'`, q)
	if publicOnly {
		db = db.Where("is_private = ?", false)
	} else if ownerID > 0 {
		if ownerKind == "" {
			ownerKind = domain.OwnerKindUser
		}
		db = db.Where("owner_kind = ? AND owner_id = ?", ownerKind, ownerID)
	} else {
		db = db.Where("is_private = ?", false)
	}

	limit, offset = clampListBounds(limit, offset)
	var models []repositoryModel
	if err := db.Order(repoOrderClause(order)).Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ExistsByOwnerFolderName(ownerKind string, ownerID uint, folderID *uint, name string) (bool, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	q := s.db.Model(&repositoryModel{}).
		Where("owner_kind = ? AND owner_id = ? AND name = ?", ownerKind, ownerID, name)
	if folderID == nil {
		q = q.Where("folder_id IS NULL")
	} else {
		q = q.Where("folder_id = ?", *folderID)
	}
	var count int64
	err := q.Count(&count).Error

	return count > 0, err
}

func (s *RepositoryStore) CountByFolderID(folderID uint) (int64, error) {
	var count int64
	err := s.db.Model(&repositoryModel{}).Where("folder_id = ?", folderID).Count(&count).Error
	return count, err
}

func (s *RepositoryStore) TouchLastActivity(id uint, at time.Time) error {
	return s.db.Model(&repositoryModel{}).Where("id = ?", id).Updates(map[string]any{
		"last_activity_at": at,
		"updated_at":       at,
	}).Error
}

func repoOrderClause(order domain.RepoOrder) string {
	if order == domain.RepoOrderName {
		return "name COLLATE NOCASE ASC, id ASC"
	}

	return "updated_at DESC"
}

func (s *RepositoryStore) CountByOwnerID(ownerKind string, ownerID uint) (int64, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	var count int64
	err := s.db.Model(&repositoryModel{}).Where("owner_kind = ? AND owner_id = ?", ownerKind, ownerID).Count(&count).Error

	return count, err
}

func repoModelsToDomain(models []repositoryModel) []domain.Repository {
	repos := make([]domain.Repository, 0, len(models))
	for _, model := range models {
		repos = append(repos, model.toDomain())
	}

	return repos
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

func (s *RepositoryStore) ListPublicIDsByOwner(ctx context.Context, ownerKind string, ownerID uint) ([]uint, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	var ids []uint
	err := s.db.WithContext(ctx).Model(&repositoryModel{}).
		Where("owner_kind = ? AND owner_id = ? AND is_private = ?", ownerKind, ownerID, false).
		Pluck("id", &ids).Error
	return ids, err
}
