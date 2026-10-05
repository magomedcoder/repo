package usecase

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrOrgNotFound       = errors.New("organization_not_found")
	ErrOrgForbidden      = errors.New("organization_forbidden")
	ErrOrgSlugRequired   = errors.New("organization_slug_required")
	ErrOrgNameRequired   = errors.New("organization_name_required")
	ErrOrgSlugTaken      = errors.New("organization_slug_taken")
	ErrInvalidOrgRole    = errors.New("invalid_organization_role")
	ErrOrgMemberExists   = errors.New("organization_member_exists")
	ErrOrgMemberNotFound = errors.New("organization_member_not_found")
	ErrCannotRemoveOwner = errors.New("cannot_remove_last_owner")
	ErrReservedSlug      = errors.New("reserved_slug")
	ErrOrgNotEmpty       = errors.New("organization_not_empty")
)

var reservedSlugs = map[string]struct{}{
	"api":           {},
	"login":         {},
	"register":      {},
	"folders":       {},
	"settings":      {},
	"search":        {},
	"orgs":          {},
	"organizations": {},
	"assets":        {},
	"static":        {},
	"users":         {},
	"me":            {},
}

type OrganizationItem struct {
	ID          uint   `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	HasAvatar   bool   `json:"has_avatar"`
	Role        string `json:"role,omitempty"`
	CreatedAt   any    `json:"created_at"`
	UpdatedAt   any    `json:"updated_at"`
}

type OrgMemberItem struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type CreateOrganizationInput struct {
	CreatorID   uint
	Slug        string
	Name        string
	Description string
}

type UpdateOrganizationInput struct {
	ViewerID    uint
	Slug        string
	Name        *string
	Description *string
}

type OrganizationUseCase struct {
	orgs  domain.OrganizationStore
	users domain.UserStore
	repos domain.RepositoryStore
}

func NewOrganizationUseCase(orgs domain.OrganizationStore, users domain.UserStore, repos domain.RepositoryStore) *OrganizationUseCase {
	return &OrganizationUseCase{
		orgs:  orgs,
		users: users,
		repos: repos,
	}
}

func (uc *OrganizationUseCase) Create(in CreateOrganizationInput) (*OrganizationItem, error) {
	if in.CreatorID == 0 {
		return nil, ErrUnauthorized
	}

	slug, err := normalizeNamespaceSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrOrgNameRequired
	}

	if err := uc.ensureSlugAvailable(slug); err != nil {
		return nil, err
	}

	org := &domain.Organization{
		Slug:        slug,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		CreatedBy:   in.CreatorID,
	}

	if err := uc.orgs.Create(org); err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}

	if err := uc.orgs.AddMember(&domain.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         in.CreatorID,
		Role:           domain.OrgRoleOwner,
	}); err != nil {
		_ = uc.orgs.Delete(org.ID)
		return nil, fmt.Errorf("add owner member: %w", err)
	}

	item := toOrgItem(org, domain.OrgRoleOwner)

	return &item, nil
}

func (uc *OrganizationUseCase) ListMine(userID uint) ([]OrganizationItem, error) {
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	orgs, err := uc.orgs.ListByUserID(userID)
	if err != nil {
		return nil, err
	}

	out := make([]OrganizationItem, 0, len(orgs))
	for i := range orgs {
		role := ""
		if m, err := uc.orgs.FindMember(orgs[i].ID, userID); err == nil {
			role = m.Role
		}

		out = append(out, toOrgItem(&orgs[i], role))
	}

	return out, nil
}

func (uc *OrganizationUseCase) Get(slug string, viewerID uint) (*OrganizationItem, error) {
	org, err := uc.orgs.FindBySlug(strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return nil, ErrOrgNotFound
	}

	role := ""
	if viewerID != 0 {
		if m, err := uc.orgs.FindMember(org.ID, viewerID); err == nil {
			role = m.Role
		}
	}

	item := toOrgItem(org, role)

	return &item, nil
}

func (uc *OrganizationUseCase) Update(in UpdateOrganizationInput) (*OrganizationItem, error) {
	org, member, err := uc.requireRole(in.Slug, in.ViewerID, domain.OrgRoleAdmin, domain.OrgRoleOwner)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrOrgNameRequired
		}

		org.Name = name
	}

	if in.Description != nil {
		org.Description = strings.TrimSpace(*in.Description)
	}

	if err := uc.orgs.Update(org); err != nil {
		return nil, err
	}

	item := toOrgItem(org, member.Role)

	return &item, nil
}

func (uc *OrganizationUseCase) Delete(slug string, viewerID uint) error {
	org, _, err := uc.requireRole(slug, viewerID, domain.OrgRoleOwner)
	if err != nil {
		return err
	}

	repos, err := uc.repos.ListByOwnerID(domain.OwnerKindOrg, org.ID)
	if err != nil {
		return err
	}

	if len(repos) > 0 {
		return ErrOrgNotEmpty
	}

	return uc.orgs.Delete(org.ID)
}

func (uc *OrganizationUseCase) ListMembers(slug string, viewerID uint) ([]OrgMemberItem, error) {
	org, err := uc.orgs.FindBySlug(strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return nil, ErrOrgNotFound
	}

	if _, err := uc.orgs.FindMember(org.ID, viewerID); err != nil {
		return nil, ErrOrgForbidden
	}

	members, err := uc.orgs.ListMembers(org.ID)
	if err != nil {
		return nil, err
	}
	out := make([]OrgMemberItem, 0, len(members))
	for _, m := range members {
		user, err := uc.users.FindByID(m.UserID)
		if err != nil {
			continue
		}

		out = append(out, OrgMemberItem{
			UserID:   m.UserID,
			Username: user.Username,
			Role:     m.Role,
		})
	}

	return out, nil
}

func (uc *OrganizationUseCase) AddMember(slug string, viewerID uint, username, role string) (*OrgMemberItem, error) {
	org, _, err := uc.requireRole(slug, viewerID, domain.OrgRoleAdmin, domain.OrgRoleOwner)
	if err != nil {
		return nil, err
	}

	role, err = normalizeOrgRole(role)
	if err != nil {
		return nil, err
	}

	if role == domain.OrgRoleOwner {
		if m, err := uc.orgs.FindMember(org.ID, viewerID); err != nil || m.Role != domain.OrgRoleOwner {
			return nil, ErrOrgForbidden
		}
	}

	user, err := uc.users.FindByUsername(strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return nil, ErrUserNotFound
	}

	if _, err := uc.orgs.FindMember(org.ID, user.ID); err == nil {
		return nil, ErrOrgMemberExists
	}

	member := &domain.OrganizationMember{OrganizationID: org.ID, UserID: user.ID, Role: role}
	if err := uc.orgs.AddMember(member); err != nil {
		return nil, err
	}

	return &OrgMemberItem{
		UserID:   user.ID,
		Username: user.Username,
		Role:     role,
	}, nil
}

func (uc *OrganizationUseCase) UpdateMemberRole(slug string, viewerID, targetUserID uint, role string) (*OrgMemberItem, error) {
	org, viewerMember, err := uc.requireRole(slug, viewerID, domain.OrgRoleOwner)
	if err != nil {
		return nil, err
	}

	_ = viewerMember
	role, err = normalizeOrgRole(role)
	if err != nil {
		return nil, err
	}

	target, err := uc.orgs.FindMember(org.ID, targetUserID)
	if err != nil {
		return nil, ErrOrgMemberNotFound
	}

	if target.Role == domain.OrgRoleOwner && role != domain.OrgRoleOwner {
		n, err := uc.orgs.CountByRole(org.ID, domain.OrgRoleOwner)
		if err != nil {
			return nil, err
		}

		if n <= 1 {
			return nil, ErrCannotRemoveOwner
		}
	}

	target.Role = role
	if err := uc.orgs.UpdateMember(target); err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(targetUserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &OrgMemberItem{
		UserID:   user.ID,
		Username: user.Username,
		Role:     role,
	}, nil
}

func (uc *OrganizationUseCase) RemoveMember(slug string, viewerID, targetUserID uint) error {
	org, viewerMember, err := uc.requireRole(slug, viewerID, domain.OrgRoleAdmin, domain.OrgRoleOwner)
	if err != nil {
		return err
	}

	target, err := uc.orgs.FindMember(org.ID, targetUserID)
	if err != nil {
		return ErrOrgMemberNotFound
	}

	if target.Role == domain.OrgRoleOwner {
		if viewerMember.Role != domain.OrgRoleOwner {
			return ErrOrgForbidden
		}

		n, err := uc.orgs.CountByRole(org.ID, domain.OrgRoleOwner)
		if err != nil {
			return err
		}

		if n <= 1 {
			return ErrCannotRemoveOwner
		}
	}

	if targetUserID != viewerID && viewerMember.Role == domain.OrgRoleAdmin && target.Role == domain.OrgRoleAdmin {
		return ErrOrgForbidden
	}

	return uc.orgs.RemoveMember(org.ID, targetUserID)
}

func (uc *OrganizationUseCase) Leave(slug string, viewerID uint) error {
	org, err := uc.orgs.FindBySlug(strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return ErrOrgNotFound
	}

	member, err := uc.orgs.FindMember(org.ID, viewerID)
	if err != nil {
		return ErrOrgMemberNotFound
	}

	if member.Role == domain.OrgRoleOwner {
		n, err := uc.orgs.CountByRole(org.ID, domain.OrgRoleOwner)
		if err != nil {
			return err
		}

		if n <= 1 {
			return ErrCannotRemoveOwner
		}
	}

	return uc.orgs.RemoveMember(org.ID, viewerID)
}

func (uc *OrganizationUseCase) ListRepos(slug string, viewerID uint) ([]RepositoryItem, error) {
	org, err := uc.orgs.FindBySlug(strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return nil, ErrOrgNotFound
	}

	repos, err := uc.repos.ListByOwnerID(domain.OwnerKindOrg, org.ID)
	if err != nil {
		return nil, err
	}

	out := make([]RepositoryItem, 0, len(repos))
	isMember := false
	if viewerID != 0 {
		if _, err := uc.orgs.FindMember(org.ID, viewerID); err == nil {
			isMember = true
		}
	}

	for i := range repos {
		if repos[i].IsPrivate && !isMember {
			continue
		}
		out = append(out, toRepositoryItem(&repos[i], org.Slug, ""))
	}

	return out, nil
}

func (uc *OrganizationUseCase) ensureSlugAvailable(slug string) error {
	if _, ok := reservedSlugs[slug]; ok {
		return ErrReservedSlug
	}

	taken, err := uc.orgs.ExistsBySlug(slug)
	if err != nil {
		return err
	}

	if taken {
		return ErrOrgSlugTaken
	}

	taken, err = uc.users.ExistsByUsername(slug)
	if err != nil {
		return err
	}

	if taken {
		return ErrOrgSlugTaken
	}

	return nil
}

func (uc *OrganizationUseCase) requireRole(slug string, viewerID uint, roles ...string) (*domain.Organization, *domain.OrganizationMember, error) {
	if viewerID == 0 {
		return nil, nil, ErrUnauthorized
	}

	org, err := uc.orgs.FindBySlug(strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return nil, nil, ErrOrgNotFound
	}

	member, err := uc.orgs.FindMember(org.ID, viewerID)
	if err != nil {
		return nil, nil, ErrOrgForbidden
	}

	if slices.Contains(roles, member.Role) {
		return org, member, nil
	}

	return nil, nil, ErrOrgForbidden
}

func normalizeNamespaceSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if slug == "" {
		return "", ErrOrgSlugRequired
	}

	if err := validateUsername(slug); err != nil {
		if errors.Is(err, ErrUsernameRequired) {
			return "", ErrOrgSlugRequired
		}

		if errors.Is(err, ErrInvalidUsername) {
			return "", ErrInvalidUsername
		}

		return "", err
	}

	if _, ok := reservedSlugs[slug]; ok {
		return "", ErrReservedSlug
	}

	return slug, nil
}

func normalizeOrgRole(role string) (string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		role = domain.OrgRoleMember
	}

	switch role {
	case domain.OrgRoleOwner, domain.OrgRoleAdmin, domain.OrgRoleMember:
		return role, nil
	default:
		return "", ErrInvalidOrgRole
	}
}

func toOrgItem(org *domain.Organization, role string) OrganizationItem {
	return OrganizationItem{
		ID:          org.ID,
		Slug:        org.Slug,
		Name:        org.Name,
		Description: org.Description,
		HasAvatar:   org.AvatarPath != "",
		Role:        role,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}
}

func orgRoleAtLeast(role string, min string) bool {
	rank := map[string]int{
		domain.OrgRoleMember: 1,
		domain.OrgRoleAdmin:  2,
		domain.OrgRoleOwner:  3,
	}

	return rank[role] >= rank[min]
}
