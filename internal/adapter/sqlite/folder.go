package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type folderModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	OwnerKind string       `gorm:"size:8;not null;default:user"`
	OwnerID   uint         `gorm:"not null"`
	ParentID  *uint        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Parent    *folderModel `gorm:"foreignKey:ParentID;references:ID"`
	Name      string       `gorm:"size:64;not null"`
	Slug      string       `gorm:"size:64;not null"`
	Path      string       `gorm:"size:512;not null"`
}

func (folderModel) TableName() string {
	return "folders"
}

func (m folderModel) toDomain() domain.Folder {
	kind := m.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	return domain.Folder{
		ID:        m.ID,
		OwnerKind: kind,
		OwnerID:   m.OwnerID,
		Name:      m.Name,
		Slug:      m.Slug,
		Path:      m.Path,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func folderFromDomain(f *domain.Folder) *folderModel {
	kind := f.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	return &folderModel{
		ID:        f.ID,
		OwnerKind: kind,
		OwnerID:   f.OwnerID,
		ParentID:  nil,
		Name:      f.Name,
		Slug:      f.Slug,
		Path:      f.Path,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
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
	err := s.db.First(&model, id).Error
	if err != nil {
		return nil, mapDBErr(err)
	}

	f := model.toDomain()
	return &f, nil
}

func (s *FolderStore) FindByPath(path string) (*domain.Folder, error) {
	var model folderModel
	err := s.db.Where("path = ?", path).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}

	f := model.toDomain()
	return &f, nil
}

func (s *FolderStore) FindByOwnerAndID(ownerKind string, ownerID, id uint) (*domain.Folder, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	var model folderModel
	err := s.db.Where("owner_kind = ? AND owner_id = ? AND id = ?", ownerKind, ownerID, id).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}

	f := model.toDomain()
	return &f, nil
}

func (s *FolderStore) FindByOwnerAndPath(ownerKind string, ownerID uint, path string) (*domain.Folder, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	var model folderModel
	err := s.db.Where("owner_kind = ? AND owner_id = ? AND path = ?", ownerKind, ownerID, path).First(&model).Error
	if err != nil {
		return nil, mapDBErr(err)
	}

	f := model.toDomain()
	return &f, nil
}

func (s *FolderStore) ListByOwner(ownerKind string, ownerID uint) ([]domain.Folder, error) {
	if ownerKind == "" {
		ownerKind = domain.OwnerKindUser
	}

	var models []folderModel
	err := s.db.Where("owner_kind = ? AND owner_id = ? AND parent_id IS NULL", ownerKind, ownerID).Order("name ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	return folderModelsToDomain(models), nil
}

func (s *FolderStore) ListForMember(userID uint) ([]domain.Folder, error) {
	var models []folderModel
	err := s.db.Model(&folderModel{}).
		Joins("JOIN folder_members ON folder_members.folder_id = folders.id AND folder_members.deleted_at IS NULL").
		Where("folder_members.user_id = ? AND folders.parent_id IS NULL", userID).
		Order("folders.name ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	return folderModelsToDomain(models), nil
}

func (s *FolderStore) ExistsBySlug(slug string) (bool, error) {
	var count int64
	err := s.db.Model(&folderModel{}).
		Where("parent_id IS NULL AND slug = ?", slug).
		Count(&count).Error
	return count > 0, err
}

func (s *FolderStore) ExistsByOwnerSlug(ownerKind string, ownerID uint, slug string) (bool, error) {
	_ = ownerKind
	_ = ownerID
	return s.ExistsBySlug(slug)
}

func folderModelsToDomain(models []folderModel) []domain.Folder {
	out := make([]domain.Folder, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out
}
