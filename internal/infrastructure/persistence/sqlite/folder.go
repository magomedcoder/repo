package sqlite

import (
	"errors"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type folderModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	OwnerID   uint           `gorm:"not null;uniqueIndex:idx_folder_scope"`
	ParentID  uint           `gorm:"not null;uniqueIndex:idx_folder_scope"` // 0 = root
	Name      string         `gorm:"size:64;not null"`
	Slug      string         `gorm:"size:64;not null;uniqueIndex:idx_folder_scope"`
	Path      string         `gorm:"size:512;not null;index"`
}

func (folderModel) TableName() string {
	return "folders"
}

func (m folderModel) toDomain() domain.Folder {
	f := domain.Folder{
		ID:        m.ID,
		OwnerID:   m.OwnerID,
		Name:      m.Name,
		Slug:      m.Slug,
		Path:      m.Path,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.ParentID != 0 {
		pid := m.ParentID
		f.ParentID = &pid
	}

	return f
}

func folderFromDomain(f *domain.Folder) *folderModel {
	m := &folderModel{
		ID:        f.ID,
		OwnerID:   f.OwnerID,
		Name:      f.Name,
		Slug:      f.Slug,
		Path:      f.Path,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}

	if f.ParentID != nil {
		m.ParentID = *f.ParentID
	}

	return m
}

type FolderStore struct {
	db *gorm.DB
}

func NewFolderStore(db *gorm.DB) *FolderStore {
	return &FolderStore{db: db}
}

func (s *FolderStore) Create(folder *domain.Folder) error {
	model := folderFromDomain(folder)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*folder = model.toDomain()
	return nil
}

func (s *FolderStore) Update(folder *domain.Folder) error {
	model := folderFromDomain(folder)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*folder = model.toDomain()
	return nil
}

func (s *FolderStore) Delete(id uint) error {
	return s.db.Delete(&folderModel{}, id).Error
}

func (s *FolderStore) FindByID(id uint) (*domain.Folder, error) {
	var model folderModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, err
	}

	f := model.toDomain()
	return &f, nil
}

func (s *FolderStore) FindByOwnerAndID(ownerID, id uint) (*domain.Folder, error) {
	var model folderModel
	err := s.db.Where("owner_id = ? AND id = ?", ownerID, id).First(&model).Error
	if err != nil {
		return nil, err
	}

	f := model.toDomain()
	return &f, nil
}

func (s *FolderStore) FindByOwnerAndPath(ownerID uint, path string) (*domain.Folder, error) {
	var model folderModel
	err := s.db.Where("owner_id = ? AND path = ?", ownerID, path).First(&model).Error
	if err != nil {
		return nil, err
	}

	f := model.toDomain()
	return &f, nil
}

func (s *FolderStore) ListByOwner(ownerID uint) ([]domain.Folder, error) {
	var models []folderModel
	err := s.db.Where("owner_id = ?", ownerID).Order("path ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return folderModelsToDomain(models), nil
}

func (s *FolderStore) ListByOwnerAndParent(ownerID uint, parentID *uint) ([]domain.Folder, error) {
	pid := uint(0)
	if parentID != nil {
		pid = *parentID
	}

	var models []folderModel
	err := s.db.Where("owner_id = ? AND parent_id = ?", ownerID, pid).Order("name ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return folderModelsToDomain(models), nil
}

func (s *FolderStore) ExistsByOwnerParentSlug(ownerID uint, parentID *uint, slug string) (bool, error) {
	pid := uint(0)
	if parentID != nil {
		pid = *parentID
	}

	var count int64
	err := s.db.Model(&folderModel{}).
		Where("owner_id = ? AND parent_id = ? AND slug = ?", ownerID, pid, slug).
		Count(&count).Error
	return count > 0, err
}

func (s *FolderStore) CountChildren(folderID uint) (int64, error) {
	var count int64
	err := s.db.Model(&folderModel{}).Where("parent_id = ?", folderID).Count(&count).Error
	return count, err
}

func (s *FolderStore) ListDescendants(ownerID uint, pathPrefix string) ([]domain.Folder, error) {
	var models []folderModel
	err := s.db.Where("owner_id = ? AND (path = ? OR path LIKE ?)", ownerID, pathPrefix, pathPrefix+"/%").
		Order("path ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	return folderModelsToDomain(models), nil
}

func folderModelsToDomain(models []folderModel) []domain.Folder {
	out := make([]domain.Folder, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
