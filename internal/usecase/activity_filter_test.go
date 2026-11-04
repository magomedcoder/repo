package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type memActivity struct {
	events []domain.ActivityEvent
}

func (s *memActivity) Create(ev *domain.ActivityEvent) error {
	s.events = append(s.events, *ev)
	return nil
}

func (s *memActivity) ListByOwner(ownerKind string, ownerID uint, limit int) ([]domain.ActivityEvent, error) {
	var out []domain.ActivityEvent
	for _, ev := range s.events {
		if ev.OwnerKind == ownerKind && ev.OwnerID == ownerID {
			out = append(out, ev)
		}
	}

	return out, nil
}

func (s *memActivity) ListPublicByOwner(ownerKind string, ownerID uint, limit int) ([]domain.ActivityEvent, error) {
	return s.ListByOwner(ownerKind, ownerID, limit)
}

type memRepos struct {
	items []domain.Repository
}

func (s *memRepos) Create(*domain.Repository) error {
	return nil
}

func (s *memRepos) Update(*domain.Repository) error {
	return nil
}

func (s *memRepos) Delete(uint) error {
	return nil
}
func (s *memRepos) FindByRepoPath(_ context.Context, repoPath string) (*domain.Repository, error) {
	for i := range s.items {
		r := s.items[i]
		if r.RepoPath == repoPath {
			cp := r
			return &cp, nil
		}
	}

	return nil, errors.New("not found")
}

func (s *memRepos) ListByOwnerID(_ context.Context, ownerKind string, ownerID uint, limit, offset int, _ domain.RepoOrder) ([]domain.Repository, error) {
	_ = limit
	_ = offset
	var out []domain.Repository
	for _, r := range s.items {
		if r.OwnerKind == ownerKind && r.OwnerID == ownerID {
			out = append(out, r)
		}
	}

	return out, nil
}

func (s *memRepos) CountByOwnerID(ownerKind string, ownerID uint) (int64, error) {
	var n int64
	for _, r := range s.items {
		if r.OwnerKind == ownerKind && r.OwnerID == ownerID {
			n++
		}
	}

	return n, nil
}

func (s *memRepos) ListPublicIDsByOwner(_ context.Context, ownerKind string, ownerID uint) ([]uint, error) {
	var out []uint
	for _, r := range s.items {
		if r.OwnerKind == ownerKind && r.OwnerID == ownerID && !r.IsPrivate {
			out = append(out, r.ID)
		}
	}

	return out, nil
}

func (s *memRepos) ListByOwnerAndFolderID(context.Context, string, uint, *uint) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ListByFolderID(context.Context, uint) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ListPublic(context.Context, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ListHome(context.Context, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) SearchByName(context.Context, string, string, uint, bool, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ExistsByOwnerFolderName(string, uint, *uint, string) (bool, error) {
	return false, nil
}

func (s *memRepos) CountByFolderID(uint) (int64, error) {
	return 0, nil
}

func (s *memRepos) TouchLastActivity(uint, time.Time) error {
	return nil
}

func TestListOwnerHidesPrivateActivity(t *testing.T) {
	t.Parallel()
	privID := uint(1)
	pubID := uint(2)
	repos := &memRepos{items: []domain.Repository{
		{
			ID:        privID,
			OwnerKind: domain.OwnerKindUser,
			OwnerID:   1,
			IsPrivate: true,
			Name:      "secret",
		},
		{
			ID:        pubID,
			OwnerKind: domain.OwnerKindUser,
			OwnerID:   1,
			IsPrivate: false,
			Name:      "open",
		},
	}}
	events := &memActivity{events: []domain.ActivityEvent{
		{
			ID: 1, OwnerKind: domain.OwnerKindUser,
			OwnerID: 1,
			ActorID: 1,
			RepoID:  &privID,
			Type:    domain.ActivityPush,
			Summary: "secret push",
		},
		{
			ID: 2, OwnerKind: domain.OwnerKindUser,
			OwnerID: 1,
			ActorID: 1,
			RepoID:  &pubID,
			Type:    domain.ActivityPush,
			Summary: "public push",
		},
	}}
	users := &memUsers{
		items: []*domain.User{
			{
				ID:       1,
				Username: "ada",
			},
		},
		next: 1,
	}
	uc := NewActivityUseCase(events, users, repos)

	anon, err := uc.ListOwner(t.Context(), "ada", 0, 30)
	if err != nil {
		t.Fatal(err)
	}

	if len(anon) != 1 || anon[0].Summary != "public push" {
		t.Fatalf("anon=%+v", anon)
	}

	owner, err := uc.ListOwner(t.Context(), "ada", 1, 30)
	if err != nil {
		t.Fatal(err)
	}

	if len(owner) != 2 {
		t.Fatalf("owner=%+v", owner)
	}
}
