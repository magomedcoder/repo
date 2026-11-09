package usecase

import (
	"context"
	"testing"

	"github.com/magomedcoder/repo/internal/domain"
)

type memTeams struct {
	items []*domain.Team
	next  uint
}

func (m *memTeams) Create(t *domain.Team) error {
	m.next++
	t.ID = m.next
	cp := *t
	m.items = append(m.items, &cp)

	return nil
}

func (m *memTeams) Update(t *domain.Team) error {
	for i := range m.items {
		if m.items[i].ID == t.ID {
			cp := *t
			m.items[i] = &cp
		}
	}

	return nil
}

func (m *memTeams) Delete(id uint) error {
	kept := m.items[:0]
	for _, t := range m.items {
		if t.ID != id {
			kept = append(kept, t)
		}
	}

	m.items = kept

	return nil
}

func (m *memTeams) FindByID(id uint) (*domain.Team, error) {
	for _, t := range m.items {
		if t.ID == id {
			cp := *t
			return &cp, nil
		}
	}

	return nil, ErrTeamNotFound
}

func (m *memTeams) FindByOwnerSlug(owner uint, slug string) (*domain.Team, error) {
	for _, t := range m.items {
		if t.OwnerUserID == owner && t.Slug == slug {
			cp := *t
			return &cp, nil
		}
	}

	return nil, ErrTeamNotFound
}

func (m *memTeams) ListByOwner(owner uint) ([]domain.Team, error) {
	var out []domain.Team
	for _, t := range m.items {
		if t.OwnerUserID == owner {
			out = append(out, *t)
		}
	}

	return out, nil
}

func (m *memTeams) ExistsByOwnerSlug(owner uint, slug string) (bool, error) {
	_, err := m.FindByOwnerSlug(owner, slug)

	return err == nil, nil
}

type memTeamMembers struct {
	rows []domain.TeamMember
}

func (m *memTeamMembers) Create(mem *domain.TeamMember) error {
	m.rows = append(m.rows, *mem)

	return nil
}

func (m *memTeamMembers) Delete(teamID, userID uint) error {
	kept := m.rows[:0]
	for _, r := range m.rows {
		if r.TeamID == teamID && r.UserID == userID {
			continue
		}

		kept = append(kept, r)
	}

	m.rows = kept

	return nil
}

func (m *memTeamMembers) DeleteByTeam(teamID uint) error {
	kept := m.rows[:0]
	for _, r := range m.rows {
		if r.TeamID != teamID {
			kept = append(kept, r)
		}
	}

	m.rows = kept

	return nil
}

func (m *memTeamMembers) Find(teamID, userID uint) (*domain.TeamMember, error) {
	for _, r := range m.rows {
		if r.TeamID == teamID && r.UserID == userID {
			cp := r
			return &cp, nil
		}
	}

	return nil, ErrTeamMemberNotFound
}

func (m *memTeamMembers) ListByTeam(teamID uint) ([]domain.TeamMember, error) {
	var out []domain.TeamMember
	for _, r := range m.rows {
		if r.TeamID == teamID {
			out = append(out, r)
		}
	}

	return out, nil
}

type teamFixture struct {
	uc      *TeamUseCase
	grants  *memTeamRepos
	teams   *memTeams
	members *memTeamMembers
	collabs *memCollabs
}

func newTeamFixture() *teamFixture {
	users := &memUsers{
		items: []*domain.User{
			{
				ID:       1,
				Username: "utest",
			},
			{
				ID:       2,
				Username: "utest2",
			},
			{
				ID:       3,
				Username: "carol",
			},
		},
		next: 3,
	}
	repos := &memRepos{items: []domain.Repository{
		{
			ID:        10,
			Name:      "mine",
			OwnerID:   1,
			RepoPath:  "mine.git",
			IsPrivate: true,
		},
		{
			ID:        11,
			Name:      "theirs",
			OwnerID:   2,
			RepoPath:  "theirs.git",
			IsPrivate: true,
		},
		{
			ID:       12,
			Name:     "open",
			OwnerID:  2,
			RepoPath: "open.git",
		},
		{
			ID:        13,
			Name:      "shared",
			OwnerID:   2,
			RepoPath:  "shared.git",
			IsPrivate: true,
		},
	}}

	collabs := &memCollabs{}
	_ = collabs.Upsert(&domain.RepoCollaborator{
		RepoID: 13,
		UserID: 1,
		Role:   domain.RepoRoleAdmin,
	})

	grants := &memTeamRepos{userTeams: map[uint][]uint{}}
	teams := &memTeams{}
	members := &memTeamMembers{}
	authz := NewAuthz(grants, collabs)

	return &teamFixture{
		uc:      NewTeamUseCase(teams, members, grants, repos, users, authz),
		grants:  grants,
		teams:   teams,
		members: members,
		collabs: collabs,
	}
}

func TestTeamCreateListSlugUniquePerOwner(t *testing.T) {
	t.Parallel()
	f := newTeamFixture()
	ctx := context.Background()

	item, err := f.uc.Create(ctx, 1, " Core Devs ")
	if err != nil {
		t.Fatal(err)
	}

	if item.Slug != "core-devs" || item.Name != "Core Devs" {
		t.Fatalf("item=%+v", item)
	}

	if _, err := f.uc.Create(ctx, 1, "core devs"); err != ErrTeamExists {
		t.Fatalf("want ErrTeamExists, got %v", err)
	}

	if _, err := f.uc.Create(ctx, 2, "core devs"); err != nil {
		t.Fatalf("same slug for a different owner must work: %v", err)
	}

	if _, err := f.uc.Create(ctx, 0, "x"); err != ErrUnauthorized {
		t.Fatalf("anon: %v", err)
	}

	if _, err := f.uc.Create(ctx, 1, "   "); err != ErrTeamNameRequired {
		t.Fatalf("empty: %v", err)
	}

	if _, err := f.uc.Create(ctx, 1, "a/b"); err != ErrInvalidTeamName {
		t.Fatalf("slash: %v", err)
	}

	list, err := f.uc.List(ctx, 1)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}

func TestTeamManagedOnlyByOwner(t *testing.T) {
	t.Parallel()
	f := newTeamFixture()
	ctx := context.Background()

	if _, err := f.uc.Create(ctx, 1, "devs"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.uc.Get(ctx, 2, "devs"); err != ErrTeamNotFound {
		t.Fatalf("stranger get: %v", err)
	}

	if _, err := f.uc.AddMember(ctx, 2, "devs", "carol"); err != ErrTeamNotFound {
		t.Fatalf("stranger add member: %v", err)
	}

	if err := f.uc.Delete(ctx, 2, "devs"); err != ErrTeamNotFound {
		t.Fatalf("stranger delete: %v", err)
	}

	if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 2,
		TeamSlug: "devs",
		RepoName: "open",
		Role:     "read",
	}); err != ErrTeamNotFound {
		t.Fatalf("stranger set repo: %v", err)
	}
}

func TestTeamMembersAndDeleteCleansUp(t *testing.T) {
	t.Parallel()
	f := newTeamFixture()
	ctx := context.Background()

	if _, err := f.uc.Create(ctx, 1, "devs"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.uc.AddMember(ctx, 1, "devs", "utest2"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.uc.AddMember(ctx, 1, "devs", "utest2"); err != ErrTeamMemberExists {
		t.Fatalf("dup member: %v", err)
	}

	if _, err := f.uc.AddMember(ctx, 1, "devs", "nobody"); err != ErrUserNotFound {
		t.Fatalf("unknown user: %v", err)
	}

	members, err := f.uc.ListMembers(ctx, 1, "devs")
	if err != nil || len(members) != 1 || members[0].Username != "utest2" {
		t.Fatalf("members=%v err=%v", members, err)
	}

	if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 1,
		TeamSlug: "devs",
		RepoName: "mine",
		Role:     "write",
	}); err != nil {
		t.Fatal(err)
	}

	if err := f.uc.RemoveMember(ctx, 1, "devs", "carol"); err != ErrTeamMemberNotFound {
		t.Fatalf("remove non-member: %v", err)
	}

	if err := f.uc.Delete(ctx, 1, "devs"); err != nil {
		t.Fatal(err)
	}

	if len(f.members.rows) != 0 || len(f.grants.grants) != 0 || len(f.teams.items) != 0 {
		t.Fatalf("delete must cascade: members=%v grants=%v teams=%v", f.members.rows, f.grants.grants, f.teams.items)
	}
}

func TestTeamGrantRequiresRepoAdmin(t *testing.T) {
	t.Parallel()
	f := newTeamFixture()
	ctx := context.Background()

	if _, err := f.uc.Create(ctx, 1, "devs"); err != nil {
		t.Fatal(err)
	}

	got, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 1,
		TeamSlug: "devs",
		RepoName: "mine.git",
		Role:     "admin",
	})
	if err != nil || got.RepoName != "mine" || got.Role != "admin" {
		t.Fatalf("own repo grant: %+v %v", got, err)
	}

	if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 1,
		TeamSlug: "devs",
		RepoName: "shared",
		Role:     "write",
	}); err != nil {
		t.Fatalf("collab admin grant: %v", err)
	}

	if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 1,
		TeamSlug: "devs",
		RepoName: "theirs",
		Role:     "read",
	}); err != ErrRepoNotFound {
		t.Fatalf("private foreign repo: %v", err)
	}

	for _, role := range []string{"read", "write", "admin"} {
		if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
			ViewerID: 1,
			TeamSlug: "devs",
			RepoName: "open",
			Role:     role,
		}); err != ErrRepoForbidden {
			t.Fatalf("public foreign repo role %s: %v", role, err)
		}
	}

	if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 1,
		TeamSlug: "devs",
		RepoName: "mine",
		Role:     "owner",
	}); err != ErrInvalidCollabRole {
		t.Fatalf("bad role: %v", err)
	}

	if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 1,
		TeamSlug: "devs",
		RepoName: "missing",
		Role:     "read",
	}); err != ErrRepoNotFound {
		t.Fatalf("missing repo: %v", err)
	}

	list, err := f.uc.ListRepos(ctx, 1, "devs")
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%v err=%v", list, err)
	}

	f.collabs.Delete(13, 1)
	if err := f.uc.RemoveRepo(ctx, 1, "devs", "shared"); err != nil {
		t.Fatal(err)
	}

	list, _ = f.uc.ListRepos(ctx, 1, "devs")
	if len(list) != 1 || list[0].RepoName != "mine" {
		t.Fatalf("after revoke: %v", list)
	}
}

func TestTeamRemoveRepoHidesPrivateExistence(t *testing.T) {
	t.Parallel()
	f := newTeamFixture()
	ctx := context.Background()

	if _, err := f.uc.Create(ctx, 1, "devs"); err != nil {
		t.Fatal(err)
	}

	if err := f.uc.RemoveRepo(ctx, 1, "devs", "theirs"); err != ErrRepoNotFound {
		t.Fatalf("private foreign remove: %v", err)
	}

	if err := f.uc.RemoveRepo(ctx, 1, "devs", "missing"); err != ErrRepoNotFound {
		t.Fatalf("missing remove: %v", err)
	}

	if err := f.uc.RemoveRepo(ctx, 1, "devs", "open"); err != ErrRepoNotFound {
		t.Fatalf("public foreign without grant/admin: %v", err)
	}
}

func TestTeamGrantGivesMemberAccess(t *testing.T) {
	t.Parallel()
	f := newTeamFixture()
	ctx := context.Background()

	if _, err := f.uc.Create(ctx, 1, "devs"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: 1,
		TeamSlug: "devs",
		RepoName: "mine",
		Role:     "write",
	}); err != nil {
		t.Fatal(err)
	}

	team, _ := f.teams.FindByOwnerSlug(1, "devs")
	f.grants.userTeams[3] = []uint{team.ID}

	repo := &domain.Repository{
		ID:        10,
		OwnerID:   1,
		IsPrivate: true,
	}
	authz := f.uc.authz
	if !authz.CanReadPrivateRepo(repo, 3) || !authz.CanWriteRepo(repo, 3) || authz.CanAdminRepo(repo, 3) {
		t.Fatal("global team grant must give write (not admin) to a member")
	}

	if authz.CanReadPrivateRepo(repo, 2) {
		t.Fatal("non-member must not get access")
	}
}
