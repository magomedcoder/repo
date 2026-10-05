package sqlite

import (
	"fmt"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type organizationModel struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Slug        string         `gorm:"uniqueIndex;size:39;not null"`
	Name        string         `gorm:"size:100;not null"`
	Description string
	AvatarPath  string `gorm:"size:255"`
	CreatedBy   uint   `gorm:"not null;index"`
}

func (organizationModel) TableName() string { 
	return "organizations" 
}

func (m organizationModel) toDomain() *domain.Organization {
	return &domain.Organization{
		ID:          m.ID,
		Slug:        m.Slug,
		Name:        m.Name,
		Description: m.Description,
		AvatarPath:  m.AvatarPath,
		CreatedBy:   m.CreatedBy,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

type organizationMemberModel struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	OrganizationID uint   `gorm:"not null;uniqueIndex:idx_org_member"`
	UserID         uint   `gorm:"not null;uniqueIndex:idx_org_member;index"`
	Role           string `gorm:"size:16;not null"`
}

func (organizationMemberModel) TableName() string { 
	return "organization_members" 
}

func (m organizationMemberModel) toDomain() *domain.OrganizationMember {
	return &domain.OrganizationMember{
		ID:             m.ID,
		OrganizationID: m.OrganizationID,
		UserID:         m.UserID,
		Role:           m.Role,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

type OrganizationStore struct {
	db *gorm.DB
}

func NewOrganizationStore(db *gorm.DB) *OrganizationStore {
	return &OrganizationStore{db: db}
}

func (s *OrganizationStore) Create(org *domain.Organization) error {
	model := &organizationModel{
		Slug:        org.Slug,
		Name:        org.Name,
		Description: org.Description,
		AvatarPath:  org.AvatarPath,
		CreatedBy:   org.CreatedBy,
	}

	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*org = *model.toDomain()

	return nil
}

func (s *OrganizationStore) Update(org *domain.Organization) error {
	return s.db.Model(&organizationModel{}).Where("id = ?", org.ID).Updates(map[string]any{
		"name":         org.Name,
		"description":  org.Description,
		"avatar_path":  org.AvatarPath,
	}).Error
}

func (s *OrganizationStore) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("organization_id = ?", id).Delete(&organizationMemberModel{}).Error; err != nil {
			return err
		}

		return tx.Delete(&organizationModel{}, id).Error
	})
}

func (s *OrganizationStore) FindByID(id uint) (*domain.Organization, error) {
	var model organizationModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, err
	}

	return model.toDomain(), nil
}

func (s *OrganizationStore) FindBySlug(slug string) (*domain.Organization, error) {
	var model organizationModel
	if err := s.db.Where("slug = ?", slug).First(&model).Error; err != nil {
		return nil, err
	}

	return model.toDomain(), nil
}

func (s *OrganizationStore) ExistsBySlug(slug string) (bool, error) {
	var count int64
	err := s.db.Model(&organizationModel{}).Where("slug = ?", slug).Count(&count).Error
	return count > 0, err
}

func (s *OrganizationStore) ListByUserID(userID uint) ([]domain.Organization, error) {
	var models []organizationModel
	err := s.db.Joins("JOIN organization_members ON organization_members.organization_id = organizations.id AND organization_members.user_id = ?", userID).
		Order("organizations.name ASC").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	out := make([]domain.Organization, 0, len(models))
	for _, m := range models {
		out = append(out, *m.toDomain())
	}

	return out, nil
}

func (s *OrganizationStore) AddMember(m *domain.OrganizationMember) error {
	model := &organizationMemberModel{
		OrganizationID: m.OrganizationID,
		UserID:         m.UserID,
		Role:           m.Role,
	}
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*m = *model.toDomain()

	return nil
}

func (s *OrganizationStore) UpdateMember(m *domain.OrganizationMember) error {
	return s.db.Model(&organizationMemberModel{}).
		Where("organization_id = ? AND user_id = ?", m.OrganizationID, m.UserID).
		Update("role", m.Role).Error
}

func (s *OrganizationStore) RemoveMember(orgID, userID uint) error {
	return s.db.Where("organization_id = ? AND user_id = ?", orgID, userID).Delete(&organizationMemberModel{}).Error
}

func (s *OrganizationStore) FindMember(orgID, userID uint) (*domain.OrganizationMember, error) {
	var model organizationMemberModel
	if err := s.db.Where("organization_id = ? AND user_id = ?", orgID, userID).First(&model).Error; err != nil {
		return nil, err
	}

	return model.toDomain(), nil
}

func (s *OrganizationStore) ListMembers(orgID uint) ([]domain.OrganizationMember, error) {
	var models []organizationMemberModel
	if err := s.db.Where("organization_id = ?", orgID).Order("role ASC, id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.OrganizationMember, 0, len(models))
	for _, m := range models {
		out = append(out, *m.toDomain())
	}

	return out, nil
}

func (s *OrganizationStore) CountByRole(orgID uint, role string) (int64, error) {
	var count int64
	err := s.db.Model(&organizationMemberModel{}).Where("organization_id = ? AND role = ?", orgID, role).Count(&count).Error
	return count, err
}

func ensureOrgTables(db *gorm.DB) error {
	if err := db.AutoMigrate(&organizationModel{}, &organizationMemberModel{}); err != nil {
		return fmt.Errorf("migrate organizations: %w", err)
	}
	
	return nil
}
