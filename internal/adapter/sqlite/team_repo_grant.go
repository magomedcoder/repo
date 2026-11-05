package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type teamRepoGrantModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	TeamID    uint   `gorm:"not null"`
	RepoID    uint   `gorm:"not null"`
	Role      string `gorm:"size:16;not null"`
}

func (teamRepoGrantModel) TableName() string {
	return "team_repo_grants"
}

func (m teamRepoGrantModel) toDomain() domain.TeamRepoGrant {
	return domain.TeamRepoGrant{
		ID:        m.ID,
		TeamID:    m.TeamID,
		RepoID:    m.RepoID,
		Role:      domain.RepoCollabRole(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type TeamRepoGrantStore struct {
	db *gorm.DB
}

func NewTeamRepoGrantStore(db *gorm.DB) *TeamRepoGrantStore {
	return &TeamRepoGrantStore{db: db}
}

func (s *TeamRepoGrantStore) Upsert(g *domain.TeamRepoGrant) error {
	var existing teamRepoGrantModel
	err := s.db.Unscoped().Where("team_id = ? AND repo_id = ?", g.TeamID, g.RepoID).First(&existing).Error
	if err == nil {
		existing.DeletedAt = gorm.DeletedAt{}
		existing.Role = string(g.Role)
		if err := s.db.Unscoped().Save(&existing).Error; err != nil {
			return err
		}
		*g = existing.toDomain()
		return nil
	}

	model := teamRepoGrantModel{
		TeamID: g.TeamID,
		RepoID: g.RepoID,
		Role:   string(g.Role),
	}

	if err := s.db.Create(&model).Error; err != nil {
		return err
	}
	*g = model.toDomain()

	return nil
}

func (s *TeamRepoGrantStore) Delete(teamID, repoID uint) error {
	return s.db.Where("team_id = ? AND repo_id = ?", teamID, repoID).Delete(&teamRepoGrantModel{}).Error
}

func (s *TeamRepoGrantStore) ListByTeam(teamID uint) ([]domain.TeamRepoGrant, error) {
	var models []teamRepoGrantModel
	if err := s.db.Where("team_id = ?", teamID).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.TeamRepoGrant, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func (s *TeamRepoGrantStore) ListByRepoUser(repoID, userID uint) ([]domain.TeamRepoGrant, error) {
	var models []teamRepoGrantModel
	err := s.db.Model(&teamRepoGrantModel{}).
		Joins("JOIN team_members ON team_members.team_id = team_repo_grants.team_id AND team_members.deleted_at IS NULL").
		Where("team_repo_grants.repo_id = ? AND team_members.user_id = ?", repoID, userID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.TeamRepoGrant, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}
