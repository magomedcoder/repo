package sqlite

import (
	"fmt"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type RepositoryStore struct {
	db *gorm.DB
}

func NewDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.AutoMigrate(&userModel{}, &sessionModel{}, &folderModel{}, &accessTokenModel{}, &sshKeyModel{}, &issueModel{}, &issueCommentModel{}, &labelModel{}, &issueLabelModel{}, &pullRequestModel{}, &pullCommentModel{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if err := migrateRepositories(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrateRepositories(db *gorm.DB) error {
	if !db.Migrator().HasTable(&repositoryModel{}) {
		if err := db.AutoMigrate(&repositoryModel{}); err != nil {
			return fmt.Errorf("migrate repositories: %w", err)
		}
		return nil
	}

	if !db.Migrator().HasColumn(&repositoryModel{}, "FolderID") {
		if err := db.Exec("ALTER TABLE repositories ADD COLUMN folder_id integer NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("add folder_id: %w", err)
		}
	}
	if !db.Migrator().HasColumn(&repositoryModel{}, "IsPrivate") {
		if err := db.Exec("ALTER TABLE repositories ADD COLUMN is_private integer NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("add is_private: %w", err)
		}
	}
	if !db.Migrator().HasColumn(&repositoryModel{}, "DefaultBranch") {
		if err := db.Exec("ALTER TABLE repositories ADD COLUMN default_branch text NOT NULL DEFAULT 'main'").Error; err != nil {
			return fmt.Errorf("add default_branch: %w", err)
		}
	}
	if !db.Migrator().HasColumn(&repositoryModel{}, "LastActivityAt") {
		if err := db.Exec("ALTER TABLE repositories ADD COLUMN last_activity_at datetime").Error; err != nil {
			return fmt.Errorf("add last_activity_at: %w", err)
		}
	}

	_ = db.Exec("DROP INDEX IF EXISTS `idx_repositories_name`").Error
	_ = db.Exec("DROP INDEX IF EXISTS `uni_repositories_name`").Error

	if err := db.AutoMigrate(&repositoryModel{}); err != nil {
		return fmt.Errorf("migrate repositories: %w", err)
	}
	return nil
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

func (s *RepositoryStore) FindByOwnerFolderName(ownerID uint, folderID *uint, name string) (*domain.Repository, error) {
	fid := uint(0)
	if folderID != nil {
		fid = *folderID
	}

	var model repositoryModel
	err := s.db.Where("owner_id = ? AND folder_id = ? AND name = ?", ownerID, fid, name).First(&model).Error
	if err != nil {
		return nil, err
	}

	repo := model.toDomain()
	return &repo, nil
}

func (s *RepositoryStore) ListByOwnerID(ownerID uint) ([]domain.Repository, error) {
	var models []repositoryModel
	err := s.db.Where("owner_id = ?", ownerID).Order("updated_at DESC").Find(&models).Error
	if err != nil {
		return nil, err
	}
	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ListByOwnerAndFolderID(ownerID uint, folderID *uint) ([]domain.Repository, error) {
	fid := uint(0)
	if folderID != nil {
		fid = *folderID
	}

	var models []repositoryModel
	err := s.db.Where("owner_id = ? AND folder_id = ?", ownerID, fid).Order("name ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ListPublic(limit int) ([]domain.Repository, error) {
	if limit <= 0 {
		limit = 100
	}

	var models []repositoryModel
	err := s.db.Where("is_private = ?", false).Order("updated_at DESC").Limit(limit).Find(&models).Error
	if err != nil {
		return nil, err
	}

	return repoModelsToDomain(models), nil
}

func (s *RepositoryStore) ExistsByOwnerFolderName(ownerID uint, folderID *uint, name string) (bool, error) {
	fid := uint(0)
	if folderID != nil {
		fid = *folderID
	}

	var count int64
	err := s.db.Model(&repositoryModel{}).
		Where("owner_id = ? AND folder_id = ? AND name = ?", ownerID, fid, name).
		Count(&count).Error
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
