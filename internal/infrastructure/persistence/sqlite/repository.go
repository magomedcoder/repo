package sqlite

import (
	"fmt"

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

	if err := db.AutoMigrate(&userModel{}, &sessionModel{}, &folderModel{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if !db.Migrator().HasTable(&repositoryModel{}) {
		if err := db.AutoMigrate(&repositoryModel{}); err != nil {
			return nil, fmt.Errorf("migrate repositories: %w", err)
		}
	} else {
		if !db.Migrator().HasColumn(&repositoryModel{}, "FolderID") {
			if err := db.Exec("ALTER TABLE repositories ADD COLUMN folder_id integer NOT NULL DEFAULT 0").Error; err != nil {
				return nil, fmt.Errorf("add folder_id: %w", err)
			}
		}
		if err := db.AutoMigrate(&repositoryModel{}); err != nil {
			return nil, fmt.Errorf("migrate repositories: %w", err)
		}
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

	repo.ID = model.ID
	repo.CreatedAt = model.CreatedAt
	repo.UpdatedAt = model.UpdatedAt
	if model.FolderID != 0 {
		fid := model.FolderID
		repo.FolderID = &fid
	} else {
		repo.FolderID = nil
	}

	return nil
}

func (s *RepositoryStore) ListByOwnerID(ownerID uint) ([]domain.Repository, error) {
	var models []repositoryModel
	err := s.db.Where("owner_id = ?", ownerID).Order("updated_at DESC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	repos := make([]domain.Repository, 0, len(models))
	for _, model := range models {
		repos = append(repos, model.toDomain())
	}

	return repos, nil
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

	repos := make([]domain.Repository, 0, len(models))
	for _, model := range models {
		repos = append(repos, model.toDomain())
	}

	return repos, nil
}

func (s *RepositoryStore) ExistsByName(name string) (bool, error) {
	var count int64
	err := s.db.Model(&repositoryModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *RepositoryStore) CountByFolderID(folderID uint) (int64, error) {
	var count int64
	err := s.db.Model(&repositoryModel{}).Where("folder_id = ?", folderID).Count(&count).Error
	return count, err
}
