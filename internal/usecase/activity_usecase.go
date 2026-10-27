package usecase

import (
	"fmt"
	"log"
	"strings"

	"git.magomedcoder.ru/repo/internal/domain"
)

type ActivityItem struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Ref       string `json:"ref,omitempty"`
	Actor     string `json:"actor"`
	Owner     string `json:"owner"`
	OwnerKind string `json:"owner_kind"`
	CreatedAt any    `json:"created_at"`
}

type ActivityUseCase struct {
	events domain.ActivityStore
	users  domain.UserStore
	repos  domain.RepositoryStore
}

func NewActivityUseCase(
	events domain.ActivityStore,
	users domain.UserStore,
	repos domain.RepositoryStore,
) *ActivityUseCase {
	return &ActivityUseCase{
		events: events,
		users:  users,
		repos:  repos,
	}
}

func (uc *ActivityUseCase) ListOwner(ownerSlug string, viewerID uint, limit int) ([]ActivityItem, error) {
	slug := strings.ToLower(strings.TrimSpace(ownerSlug))
	kind, id, err := uc.resolveOwner(slug)
	if err != nil {
		return nil, err
	}

	events, err := uc.events.ListByOwner(kind, id, limit)
	if err != nil {
		return nil, err
	}

	events, err = uc.filterVisible(events, kind, id, viewerID)
	if err != nil {
		return nil, err
	}

	return uc.mapEvents(events)
}

func (uc *ActivityUseCase) filterVisible(events []domain.ActivityEvent, ownerKind string, ownerID, viewerID uint) ([]domain.ActivityEvent, error) {
	isOwner := viewerID != 0 && ownerKind == domain.OwnerKindUser && viewerID == ownerID
	if isOwner {
		return events, nil
	}

	repos, err := uc.repos.ListByOwnerID(ownerKind, ownerID, 0, 0)
	if err != nil {
		return nil, err
	}

	public := make(map[uint]struct{}, len(repos))
	for _, repo := range repos {
		if !repo.IsPrivate {
			public[repo.ID] = struct{}{}
		}
	}

	out := make([]domain.ActivityEvent, 0, len(events))
	for _, ev := range events {
		if ev.RepoID == nil {
			continue
		}

		if _, ok := public[*ev.RepoID]; ok {
			out = append(out, ev)
		}
	}

	return out, nil
}

func (uc *ActivityUseCase) resolveOwner(slug string) (string, uint, error) {
	if u, err := uc.users.FindByUsername(slug); err == nil {
		return domain.OwnerKindUser, u.ID, nil
	}

	return "", 0, ErrRepoNotFound
}

func (uc *ActivityUseCase) mapEvents(events []domain.ActivityEvent) ([]ActivityItem, error) {
	idSet := make(map[uint]struct{})
	for _, ev := range events {
		if ev.ActorID != 0 {
			idSet[ev.ActorID] = struct{}{}
		}
		if ev.OwnerID != 0 {
			idSet[ev.OwnerID] = struct{}{}
		}
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	usersByID, err := uc.users.FindByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("load activity users: %w", err)
	}

	out := make([]ActivityItem, 0, len(events))
	for _, ev := range events {
		actor := ""
		if u := usersByID[ev.ActorID]; u != nil {
			actor = u.Username
		}

		owner := ""
		if u := usersByID[ev.OwnerID]; u != nil {
			owner = u.Username
		}

		out = append(out, ActivityItem{
			ID:        ev.ID,
			Type:      ev.Type,
			Summary:   ev.Summary,
			Ref:       ev.Ref,
			Actor:     actor,
			Owner:     owner,
			OwnerKind: ev.OwnerKind,
			CreatedAt: ev.CreatedAt,
		})
	}

	return out, nil
}

func recordActivity(store domain.ActivityStore, ev *domain.ActivityEvent) {
	if store == nil || ev == nil || ev.ActorID == 0 {
		return
	}

	if err := store.Create(ev); err != nil {
		log.Printf("activity: create event: %v", err)
	}
}

func activitySummary(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func recordRepoActivity(store domain.ActivityStore, repo *domain.Repository, actorID uint, typ, summary string) {
	if repo == nil {
		return
	}

	kind := repo.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	recordActivity(store, &domain.ActivityEvent{
		ActorID:   actorID,
		OwnerKind: kind,
		OwnerID:   repo.OwnerID,
		RepoID:    &repo.ID,
		Type:      typ,
		Summary:   summary,
	})
}
