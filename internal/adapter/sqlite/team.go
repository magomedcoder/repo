package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type teamModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	FolderID  uint   `gorm:"not null"`
	Name      string `gorm:"size:64;not null"`
	Slug      string `gorm:"size:64;not null"`
	Caps      uint32 `gorm:"not null;default:0"`
}

func (teamModel) TableName() string {
	return "teams"
}

func (m teamModel) toDomain() domain.Team {
	return domain.Team{
		ID:        m.ID,
		FolderID:  m.FolderID,
		Name:      m.Name,
		Slug:      m.Slug,
		Caps:      domain.FolderCap(m.Caps),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type TeamStore struct {
	db *gorm.DB
}

func NewTeamStore(db *gorm.DB) *TeamStore {
	return &TeamStore{db: db}
}

func (s *TeamStore) Create(t *domain.Team) error {
	model := teamModel{
		FolderID: t.FolderID,
		Name:     t.Name,
		Slug:     t.Slug,
		Caps:     uint32(t.Caps),
	}
	if err := s.db.Create(&model).Error; err != nil {
		return err
	}
	*t = model.toDomain()

	return nil
}

func (s *TeamStore) Update(t *domain.Team) error {
	model := teamModel{
		ID:       t.ID,
		FolderID: t.FolderID,
		Name:     t.Name,
		Slug:     t.Slug,
		Caps:     uint32(t.Caps),
	}
	if err := s.db.Save(&model).Error; err != nil {
		return err
	}
	*t = model.toDomain()

	return nil
}

func (s *TeamStore) Delete(id uint) error {
	return s.db.Delete(&teamModel{}, id).Error
}

func (s *TeamStore) FindByID(id uint) (*domain.Team, error) {
	var model teamModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, mapDBErr(err)
	}
	out := model.toDomain()

	return &out, nil
}

func (s *TeamStore) FindByFolderSlug(folderID uint, slug string) (*domain.Team, error) {
	var model teamModel
	err := s.db.Where("folder_id = ? AND slug = ?", folderID, slug).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := model.toDomain()

	return &out, nil
}

func (s *TeamStore) ListByFolder(folderID uint) ([]domain.Team, error) {
	var models []teamModel
	if err := s.db.Where("folder_id = ?", folderID).Order("name ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.Team, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func (s *TeamStore) ExistsByFolderSlug(folderID uint, slug string) (bool, error) {
	var count int64
	err := s.db.Model(&teamModel{}).Where("folder_id = ? AND slug = ?", folderID, slug).Count(&count).Error
	return count > 0, err
}
