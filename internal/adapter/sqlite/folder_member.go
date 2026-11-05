package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type folderMemberModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	FolderID  uint   `gorm:"not null"`
	UserID    uint   `gorm:"not null"`
	Role      string `gorm:"size:16;not null"`
	Caps      uint32 `gorm:"not null;default:0"`
}

func (folderMemberModel) TableName() string {
	return "folder_members"
}

func (m folderMemberModel) toDomain() domain.FolderMember {
	return domain.FolderMember{
		ID:        m.ID,
		FolderID:  m.FolderID,
		UserID:    m.UserID,
		Role:      domain.FolderRole(m.Role),
		Caps:      domain.FolderCap(m.Caps),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type FolderMemberStore struct {
	db *gorm.DB
}

func NewFolderMemberStore(db *gorm.DB) *FolderMemberStore {
	return &FolderMemberStore{db: db}
}

func (s *FolderMemberStore) Create(m *domain.FolderMember) error {
	model := folderMemberModel{
		FolderID: m.FolderID,
		UserID:   m.UserID,
		Role:     string(m.Role),
		Caps:     uint32(m.Caps),
	}
	if err := s.db.Create(&model).Error; err != nil {
		return err
	}
	*m = model.toDomain()

	return nil
}

func (s *FolderMemberStore) Update(m *domain.FolderMember) error {
	model := folderMemberModel{
		ID:       m.ID,
		FolderID: m.FolderID,
		UserID:   m.UserID,
		Role:     string(m.Role),
		Caps:     uint32(m.Caps),
	}
	if err := s.db.Save(&model).Error; err != nil {
		return err
	}
	*m = model.toDomain()

	return nil
}

func (s *FolderMemberStore) Delete(folderID, userID uint) error {
	return s.db.Where("folder_id = ? AND user_id = ?", folderID, userID).Delete(&folderMemberModel{}).Error
}

func (s *FolderMemberStore) Find(folderID, userID uint) (*domain.FolderMember, error) {
	var model folderMemberModel
	err := s.db.Where("folder_id = ? AND user_id = ?", folderID, userID).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := model.toDomain()

	return &out, nil
}

func (s *FolderMemberStore) ListByFolder(folderID uint) ([]domain.FolderMember, error) {
	var models []folderMemberModel
	if err := s.db.Where("folder_id = ?", folderID).Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.FolderMember, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}
