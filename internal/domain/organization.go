package domain

import "time"

const (
	OwnerKindUser = "user"
	OwnerKindOrg  = "org"

	OrgRoleOwner  = "owner"
	OrgRoleAdmin  = "admin"
	OrgRoleMember = "member"
)

type Organization struct {
	ID          uint
	Slug        string
	Name        string
	Description string
	AvatarPath  string
	CreatedBy   uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OrganizationMember struct {
	ID             uint
	OrganizationID uint
	UserID         uint
	Role           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrganizationStore interface {
	Create(org *Organization) error

	Update(org *Organization) error

	Delete(id uint) error

	FindByID(id uint) (*Organization, error)

	FindBySlug(slug string) (*Organization, error)

	ExistsBySlug(slug string) (bool, error)

	ListByUserID(userID uint) ([]Organization, error)

	AddMember(m *OrganizationMember) error

	UpdateMember(m *OrganizationMember) error

	RemoveMember(orgID, userID uint) error

	FindMember(orgID, userID uint) (*OrganizationMember, error)
	
	ListMembers(orgID uint) ([]OrganizationMember, error)
	
	CountByRole(orgID uint, role string) (int64, error)
}
