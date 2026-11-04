package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type repoCollaboratorModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	RepoID    uint   `gorm:"not null"`
	UserID    uint   `gorm:"not null"`
	Role      string `gorm:"size:16;not null"`
}

func (repoCollaboratorModel) TableName() string {
	return "repo_collaborators"
}

func (m repoCollaboratorModel) toDomain() domain.RepoCollaborator {
	return domain.RepoCollaborator{
		ID:        m.ID,
		RepoID:    m.RepoID,
		UserID:    m.UserID,
		Role:      domain.RepoCollabRole(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type RepoCollaboratorStore struct {
	db *gorm.DB
}

func NewRepoCollaboratorStore(db *gorm.DB) *RepoCollaboratorStore {
	return &RepoCollaboratorStore{db: db}
}

func (s *RepoCollaboratorStore) Upsert(c *domain.RepoCollaborator) error {
	var existing repoCollaboratorModel
	err := s.db.Unscoped().Where("repo_id = ? AND user_id = ?", c.RepoID, c.UserID).First(&existing).Error
	if err == nil {
		existing.DeletedAt = gorm.DeletedAt{}
		existing.Role = string(c.Role)
		if err := s.db.Unscoped().Save(&existing).Error; err != nil {
			return err
		}
		*c = existing.toDomain()
		return nil
	}

	model := repoCollaboratorModel{
		RepoID: c.RepoID,
		UserID: c.UserID,
		Role:   string(c.Role),
	}
	if err := s.db.Create(&model).Error; err != nil {
		return err
	}

	*c = model.toDomain()

	return nil
}

func (s *RepoCollaboratorStore) Delete(repoID, userID uint) error {
	return s.db.Where("repo_id = ? AND user_id = ?", repoID, userID).Delete(&repoCollaboratorModel{}).Error
}

func (s *RepoCollaboratorStore) Find(repoID, userID uint) (*domain.RepoCollaborator, error) {
	var model repoCollaboratorModel
	err := s.db.Where("repo_id = ? AND user_id = ?", repoID, userID).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}

	out := model.toDomain()

	return &out, nil
}

func (s *RepoCollaboratorStore) ListByRepo(repoID uint) ([]domain.RepoCollaborator, error) {
	var models []repoCollaboratorModel
	if err := s.db.Where("repo_id = ?", repoID).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.RepoCollaborator, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}
