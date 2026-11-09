package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type memActivity struct {
	events []domain.ActivityEvent
}

func (s *memActivity) Create(ev *domain.ActivityEvent) error {
	s.events = append(s.events, *ev)
	return nil
}

func (s *memActivity) ListByOwner(ownerID uint, limit int) ([]domain.ActivityEvent, error) {
	var out []domain.ActivityEvent
	for _, ev := range s.events {
		if ev.OwnerID == ownerID {
			out = append(out, ev)
		}
	}

	return out, nil
}

func (s *memActivity) ListPublicByOwner(ownerID uint, limit int) ([]domain.ActivityEvent, error) {
	return s.ListByOwner(ownerID, limit)
}

func (s *memActivity) DeleteOlderThan(_ context.Context, before time.Time) (int64, error) {
	kept := s.events[:0]
	var n int64
	for _, ev := range s.events {
		if ev.CreatedAt.Before(before) {
			n++
			continue
		}

		kept = append(kept, ev)
	}

	s.events = kept

	return n, nil
}

func (s *memActivity) DeleteByRepo(_ context.Context, repoID uint) error {
	kept := s.events[:0]
	for _, ev := range s.events {
		if ev.RepoID != nil && *ev.RepoID == repoID {
			continue
		}

		kept = append(kept, ev)
	}
	s.events = kept

	return nil
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

func (s *memRepos) ListByOwnerID(_ context.Context, ownerID uint, limit, offset int, _ domain.RepoOrder) ([]domain.Repository, error) {
	_ = limit
	_ = offset
	var out []domain.Repository
	for _, r := range s.items {
		if r.OwnerID == ownerID {
			out = append(out, r)
		}
	}

	return out, nil
}

func (s *memRepos) CountByOwnerID(ownerID uint) (int64, error) {
	var n int64
	for _, r := range s.items {
		if r.OwnerID == ownerID {
			n++
		}
	}

	return n, nil
}

func (s *memRepos) ListIDsByOwner(_ context.Context, ownerID uint) ([]uint, error) {
	var out []uint
	for _, r := range s.items {
		if r.OwnerID == ownerID && !r.IsPrivate {
			out = append(out, r.ID)
		}
	}

	return out, nil
}

func (s *memRepos) ListPublic(context.Context, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ListHome(context.Context, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ListSharedForUser(context.Context, uint, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ListMirrors(context.Context, domain.MirrorListFilter) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ResetStaleMirrorSyncing(context.Context) (int64, error) { return 0, nil }

func (s *memRepos) ListMirrorSchedules(context.Context, domain.MirrorScheduleFilter) ([]domain.MirrorSchedule, error) {
	return nil, nil
}

func (s *memRepos) CountMirrorsInError(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (s *memRepos) ListAdmin(context.Context, domain.AdminRepoFilter) ([]domain.Repository, int, error) {
	return nil, 0, nil
}

func (s *memRepos) CountAll(context.Context) (int64, error) {
	return 0, nil
}

func (s *memRepos) UpdateMirrorState(context.Context, *domain.Repository) error {
	return nil
}

func (s *memRepos) FindByID(_ context.Context, id uint) (*domain.Repository, error) {
	for i := range s.items {
		if s.items[i].ID == id {
			repo := s.items[i]
			return &repo, nil
		}
	}

	return nil, errors.New("not found")
}

func (s *memRepos) SearchByName(context.Context, string, uint, bool, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) SearchAccessible(context.Context, string, uint, int, int, domain.RepoOrder) ([]domain.Repository, error) {
	return nil, nil
}

func (s *memRepos) ExistsByName(string) (bool, error) {
	return false, nil
}

func (s *memRepos) PurgeSoftDeleted(context.Context, time.Time) ([]string, []uint, error) {
	return nil, nil, nil
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
			OwnerID:   1,
			IsPrivate: true,
			Name:      "secret",
		},
		{
			ID:        pubID,
			OwnerID:   1,
			IsPrivate: false,
			Name:      "open",
		},
	}}
	events := &memActivity{events: []domain.ActivityEvent{
		{
			ID:      1,
			OwnerID: 1,
			ActorID: 1,
			RepoID:  &privID,
			Type:    domain.ActivityPush,
			Summary: "secret push",
		},
		{
			ID:      2,
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
