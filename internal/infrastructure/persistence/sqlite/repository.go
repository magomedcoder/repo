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

	if err := db.AutoMigrate(&repositoryModel{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
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

	return nil
}

func (s *RepositoryStore) ExistsByName(name string) (bool, error) {
	var count int64
	err := s.db.Model(&repositoryModel{}).Where("name = ?", name).Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
