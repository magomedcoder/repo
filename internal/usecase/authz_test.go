package usecase

import (
	"fmt"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

type memMembers struct {
	byKey map[string]*domain.FolderMember
}

func (m *memMembers) key(folderID, userID uint) string {
	return fmt.Sprintf("%d:%d", folderID, userID)
}

func (m *memMembers) Create(mem *domain.FolderMember) error {
	if m.byKey == nil {
		m.byKey = map[string]*domain.FolderMember{}
	}
	cp := *mem
	m.byKey[m.key(mem.FolderID, mem.UserID)] = &cp

	return nil
}
func (m *memMembers) Update(mem *domain.FolderMember) error {
	return m.Create(mem)
}

func (m *memMembers) Delete(folderID, userID uint) error {
	delete(m.byKey, m.key(folderID, userID))
	return nil
}

func (m *memMembers) Find(folderID, userID uint) (*domain.FolderMember, error) {
	if v, ok := m.byKey[m.key(folderID, userID)]; ok {
		cp := *v
		return &cp, nil
	}

	return nil, ErrFolderNotFound
}

func (m *memMembers) ListByFolder(uint) ([]domain.FolderMember, error) {
	return nil, nil
}

func (m *memMembers) ListFolderIDsByUser(uint) ([]uint, error) {
	return nil, nil
}

type memCollabs struct {
	byKey map[string]*domain.RepoCollaborator
}

func (m *memCollabs) key(repoID, userID uint) string {
	return fmt.Sprintf("%d:%d", repoID, userID)
}

func (m *memCollabs) Upsert(c *domain.RepoCollaborator) error {
	if m.byKey == nil {
		m.byKey = map[string]*domain.RepoCollaborator{}
	}
	cp := *c
	m.byKey[m.key(c.RepoID, c.UserID)] = &cp

	return nil
}

func (m *memCollabs) Delete(repoID, userID uint) error {
	delete(m.byKey, m.key(repoID, userID))

	return nil
}
func (m *memCollabs) Find(repoID, userID uint) (*domain.RepoCollaborator, error) {
	if v, ok := m.byKey[m.key(repoID, userID)]; ok {
		cp := *v
		return &cp, nil
	}

	return nil, ErrRepoNotFound
}

func (m *memCollabs) ListByRepo(uint) ([]domain.RepoCollaborator, error) { return nil, nil }

type memTeamRepos struct {
	grants    []domain.TeamRepoGrant
	userTeams map[uint][]uint
}

func (m *memTeamRepos) Upsert(g *domain.TeamRepoGrant) error {
	m.grants = append(m.grants, *g)
	return nil
}

func (m *memTeamRepos) Delete(uint, uint) error {
	return nil
}

func (m *memTeamRepos) Find(uint, uint) (*domain.TeamRepoGrant, error) {
	return nil, ErrRepoNotFound
}

func (m *memTeamRepos) ListByTeam(uint) ([]domain.TeamRepoGrant, error) {
	return nil, nil
}

func (m *memTeamRepos) ListByRepoUser(repoID, userID uint) ([]domain.TeamRepoGrant, error) {
	teams := m.userTeams[userID]
	var out []domain.TeamRepoGrant
	for _, g := range m.grants {
		if g.RepoID != repoID {
			continue
		}

		for _, tid := range teams {
			if tid == g.TeamID {
				out = append(out, g)
			}
		}
	}

	return out, nil
}

func TestCapsForRolePresets(t *testing.T) {
	t.Parallel()
	if domain.CapsForRole(domain.FolderRoleOwner) != domain.CapAll {
		t.Fatalf("owner caps")
	}

	admin := domain.CapsForRole(domain.FolderRoleAdmin)
	if admin&domain.CapDeleteFolder != 0 {
		t.Fatalf("admin must not delete folder")
	}

	if admin&domain.CapManageMembers == 0 {
		t.Fatalf("admin manage members")
	}

	if domain.CapsForRole(domain.FolderRoleMember) != domain.CapView {
		t.Fatalf("member view only")
	}
}

func TestAuthzPersonalOwner(t *testing.T) {
	t.Parallel()
	a := NewAuthz(nil, nil, nil, nil, nil)
	repo := &domain.Repository{
		OwnerKind: domain.OwnerKindUser,
		OwnerID:   7,
		IsPrivate: true,
	}
	if err := a.CanReadRepo(repo, 7); err != nil {
		t.Fatal(err)
	}

	if !a.CanWriteRepo(repo, 7) || !a.CanAdminRepo(repo, 7) {
		t.Fatal("owner write/admin")
	}

	if a.CanReadPrivateRepo(repo, 8) {
		t.Fatal("stranger")
	}
}

func TestAuthzPublicReadable(t *testing.T) {
	t.Parallel()
	a := NewAuthz(nil, nil, nil, nil, nil)
	repo := &domain.Repository{IsPrivate: false}
	if err := a.CanReadRepo(repo, 0); err != nil {
		t.Fatal(err)
	}
}

func TestAuthzCollaboratorRoles(t *testing.T) {
	t.Parallel()
	collabs := &memCollabs{}
	_ = collabs.Upsert(&domain.RepoCollaborator{
		RepoID: 1,
		UserID: 2,
		Role:   domain.RepoRoleRead,
	})
	_ = collabs.Upsert(&domain.RepoCollaborator{
		RepoID: 1,
		UserID: 3,
		Role:   domain.RepoRoleWrite,
	})
	_ = collabs.Upsert(&domain.RepoCollaborator{
		RepoID: 1,
		UserID: 4,
		Role:   domain.RepoRoleAdmin,
	})
	a := NewAuthz(nil, nil, nil, nil, collabs)
	repo := &domain.Repository{
		ID:        1,
		OwnerKind: domain.OwnerKindUser,
		OwnerID:   99,
		IsPrivate: true,
	}

	if !a.CanReadPrivateRepo(repo, 2) {
		t.Fatal("read collab")
	}

	if a.CanWriteRepo(repo, 2) || a.CanAdminRepo(repo, 2) {
		t.Fatal("read cannot write")
	}

	if !a.CanWriteRepo(repo, 3) || a.CanAdminRepo(repo, 3) {
		t.Fatal("write collab")
	}

	if !a.CanAdminRepo(repo, 4) {
		t.Fatal("admin collab")
	}
}

func TestAuthzTeamRepoGrant(t *testing.T) {
	t.Parallel()
	grants := &memTeamRepos{
		userTeams: map[uint][]uint{5: {10}},
		grants: []domain.TeamRepoGrant{
			{
				TeamID: 10,
				RepoID: 1,
				Role:   domain.RepoRoleWrite,
			},
		},
	}

	a := NewAuthz(nil, nil, nil, grants, nil)
	repo := &domain.Repository{
		ID:        1,
		OwnerKind: domain.OwnerKindUser,
		OwnerID:   99,
		IsPrivate: true,
	}
	if !a.CanReadPrivateRepo(repo, 5) || !a.CanWriteRepo(repo, 5) {
		t.Fatal("team grant write")
	}

	if a.CanAdminRepo(repo, 5) {
		t.Fatal("write grant is not admin")
	}

	if a.CanReadPrivateRepo(repo, 6) {
		t.Fatal("non-member")
	}
}

func TestAuthzFolderCapViewDoesNotGrantRepoRead(t *testing.T) {
	t.Parallel()
	members := &memMembers{}
	_ = members.Create(&domain.FolderMember{
		FolderID: 1,
		UserID:   2,
		Role:     domain.FolderRoleMember,
		Caps:     domain.CapView | domain.CapCreateRepos,
	})

	a := NewAuthz(members, nil, nil, nil, nil)
	repo := &domain.Repository{
		ID:        1,
		OwnerKind: domain.OwnerKindUser,
		OwnerID:   99,
		IsPrivate: true,
		FolderID:  new(uint(1)),
	}

	if a.CanReadPrivateRepo(repo, 2) {
		t.Fatal("folder view must not grant private repo read")
	}

	if !a.HasFolderCap(1, 2, domain.CapCreateRepos) {
		t.Fatal("create repos cap")
	}
}

func TestFolderCapsUnionTeams(t *testing.T) {
	t.Parallel()
	members := &memMembers{}
	_ = members.Create(&domain.FolderMember{
		FolderID: 1,
		UserID:   2,
		Role:     domain.FolderRoleMember,
		Caps:     domain.CapView,
	})

	teams := &memTeams{items: map[uint]*domain.Team{
		9: {
			ID:       9,
			FolderID: 1,
			Caps:     domain.CapCreateRepos,
		},
	}}

	tm := &memTeamMembers{userTeams: map[string][]uint{"1:2": {9}}}
	a := NewAuthz(members, teams, tm, nil, nil)
	caps, role, err := a.FolderCaps(1, 2)
	if err != nil {
		t.Fatal(err)
	}

	if role != domain.FolderRoleMember {
		t.Fatalf("role %s", role)
	}

	if caps&domain.CapView == 0 || caps&domain.CapCreateRepos == 0 {
		t.Fatalf("union caps %#x", caps)
	}
}

type memTeams struct {
	items map[uint]*domain.Team
}

func (m *memTeams) Create(*domain.Team) error {
	return nil
}

func (m *memTeams) Update(*domain.Team) error {
	return nil
}

func (m *memTeams) Delete(uint) error {
	return nil
}

func (m *memTeams) FindByID(id uint) (*domain.Team, error) {
	if t, ok := m.items[id]; ok {
		cp := *t
		return &cp, nil
	}

	return nil, ErrTeamNotFound
}

func (m *memTeams) FindByFolderSlug(uint, string) (*domain.Team, error) {
	return nil, ErrTeamNotFound
}

func (m *memTeams) ListByFolder(uint) ([]domain.Team, error) {
	return nil, nil
}

func (m *memTeams) ExistsByFolderSlug(uint, string) (bool, error) {
	return false, nil
}

type memTeamMembers struct {
	userTeams map[string][]uint
}

func (m *memTeamMembers) key(folderID, userID uint) string {
	return fmt.Sprintf("%d:%d", folderID, userID)
}

func (m *memTeamMembers) Create(*domain.TeamMember) error {
	return nil
}

func (m *memTeamMembers) Delete(uint, uint) error {
	return nil
}

func (m *memTeamMembers) Find(uint, uint) (*domain.TeamMember, error) {
	return nil, ErrTeamMemberNotFound
}

func (m *memTeamMembers) ListByTeam(uint) ([]domain.TeamMember, error) {
	return nil, nil
}

func (m *memTeamMembers) ListTeamIDsByFolderUser(folderID, userID uint) ([]uint, error) {
	return m.userTeams[m.key(folderID, userID)], nil
}
