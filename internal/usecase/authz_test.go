package usecase

import (
	"fmt"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

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

func (m *memTeamRepos) Delete(teamID, repoID uint) error {
	kept := m.grants[:0]
	for _, g := range m.grants {
		if g.TeamID == teamID && g.RepoID == repoID {
			continue
		}

		kept = append(kept, g)
	}

	m.grants = kept

	return nil
}

func (m *memTeamRepos) DeleteByTeam(teamID uint) error {
	kept := m.grants[:0]
	for _, g := range m.grants {
		if g.TeamID != teamID {
			kept = append(kept, g)
		}
	}

	m.grants = kept

	return nil
}

func (m *memTeamRepos) ListByTeam(teamID uint) ([]domain.TeamRepoGrant, error) {
	var out []domain.TeamRepoGrant
	for _, g := range m.grants {
		if g.TeamID == teamID {
			out = append(out, g)
		}
	}

	return out, nil
}

func (m *memTeamRepos) ListByRepo(repoID uint) ([]domain.TeamRepoGrant, error) {
	var out []domain.TeamRepoGrant
	for _, g := range m.grants {
		if g.RepoID == repoID {
			out = append(out, g)
		}
	}

	return out, nil
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

func TestAuthzPersonalOwner(t *testing.T) {
	t.Parallel()
	a := NewAuthz(nil, nil)
	repo := &domain.Repository{
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
	a := NewAuthz(nil, nil)
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
	a := NewAuthz(nil, collabs)
	repo := &domain.Repository{
		ID:        1,
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

	a := NewAuthz(grants, nil)
	repo := &domain.Repository{
		ID:        1,
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
