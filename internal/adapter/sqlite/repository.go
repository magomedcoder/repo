package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.magomedcoder.ru/repo/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type RepositoryStore struct {
	db *gorm.DB
}

func NewDB(dsn string) (*gorm.DB, error) {
	file := dsn
	if i := strings.Index(file, "?"); i >= 0 {
		file = file[:i]
	}

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

	if err := migrate(db); err != nil {
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

func (s *RepositoryStore) FindByRepoPath(repoPath string) (*domain.Repository, error) {
	var model repositoryModel
	err := s.db.Where("repo_path = ?", repoPath).First(&model).Error
	if err != nil {
		return nil, err
	}

	repo := model.toDomain()
	return &repo, nil
}

func (s *RepositoryStore) ListByOwnerID(ownerKind string, ownerID uint, limit, offset int) ([]domain.Repository, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	var models []repositoryModel
	q := s.db.Where("owner_kind = ? AND owner_id = ?", ownerKind, ownerID).Order("updated_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}

	if offset > 0 {
		q = q.Offset(offset)
	}

	err := q.Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ListByOwnerAndFolderID(ownerKind string, ownerID uint, folderID *uint) ([]domain.Repository, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}
	var models []repositoryModel
	q := s.db.Where("owner_kind = ? AND owner_id = ?", ownerKind, ownerID)
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

func (s *RepositoryStore) ListPublic(limit, offset int) ([]domain.Repository, error) {
	var models []repositoryModel
	q := s.db.Where("is_private = ?", false).Order("updated_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}

	if offset > 0 {
		q = q.Offset(offset)
	}

	err := q.Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) SearchByName(query string, ownerKind string, ownerID uint, publicOnly bool, limit, offset int) ([]domain.Repository, error) {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query)
	q := "%" + escaped + "%"
	db := s.db.Model(&repositoryModel{}).Where(`name LIKE ? ESCAPE '\'`, q)
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

	var models []repositoryModel
	stmt := db.Order("updated_at DESC")
	if limit > 0 {
		stmt = stmt.Limit(limit)
	}
	
	if offset > 0 {
		stmt = stmt.Offset(offset)
	}

	if err := stmt.Find(&models).Error; err != nil {
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

func repoModelsToDomain(models []repositoryModel) []domain.Repository {
	repos := make([]domain.Repository, 0, len(models))
	for _, model := range models {
		repos = append(repos, model.toDomain())
	}

	return repos
}
