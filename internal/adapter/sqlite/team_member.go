package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type teamMemberModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
	TeamID    uint `gorm:"not null"`
	UserID    uint `gorm:"not null"`
}

func (teamMemberModel) TableName() string {
	return "team_members"
}

func (m teamMemberModel) toDomain() domain.TeamMember {
	return domain.TeamMember{
		ID:        m.ID,
		TeamID:    m.TeamID,
		UserID:    m.UserID,
		CreatedAt: m.CreatedAt,
	}
}

type TeamMemberStore struct {
	db *gorm.DB
}

func NewTeamMemberStore(db *gorm.DB) *TeamMemberStore {
	return &TeamMemberStore{db: db}
}

func (s *TeamMemberStore) Create(m *domain.TeamMember) error {
	model := teamMemberModel{TeamID: m.TeamID, UserID: m.UserID}
	if err := s.db.Create(&model).Error; err != nil {
		return err
	}
	*m = model.toDomain()

	return nil
}

func (s *TeamMemberStore) Delete(teamID, userID uint) error {
	return s.db.Where("team_id = ? AND user_id = ?", teamID, userID).Delete(&teamMemberModel{}).Error
}

func (s *TeamMemberStore) Find(teamID, userID uint) (*domain.TeamMember, error) {
	var model teamMemberModel
	err := s.db.Where("team_id = ? AND user_id = ?", teamID, userID).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := model.toDomain()

	return &out, nil
}

func (s *TeamMemberStore) ListByTeam(teamID uint) ([]domain.TeamMember, error) {
	var models []teamMemberModel
	if err := s.db.Where("team_id = ?", teamID).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.TeamMember, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func (s *TeamMemberStore) ListTeamIDsByFolderUser(folderID, userID uint) ([]uint, error) {
	var ids []uint
	err := s.db.Model(&teamMemberModel{}).
		Joins("JOIN teams ON teams.id = team_members.team_id AND teams.deleted_at IS NULL").
		Where("team_members.user_id = ? AND teams.folder_id = ?", userID, folderID).
		Pluck("team_members.team_id", &ids).Error

	return ids, err
}
