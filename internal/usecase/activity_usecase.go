package usecase

import (
	"fmt"
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

func (uc *ActivityUseCase) ListMine(userID uint, limit int) ([]ActivityItem, error) {
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	events, err := uc.events.ListForUser(userID, limit)
	if err != nil {
		return nil, err
	}

	return uc.mapEvents(events)
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

	return uc.mapEvents(events)
}

func (uc *ActivityUseCase) resolveOwner(slug string) (string, uint, error) {
	if u, err := uc.users.FindByUsername(slug); err == nil {
		return domain.OwnerKindUser, u.ID, nil
	}

	return "", 0, ErrRepoNotFound
}

func (uc *ActivityUseCase) mapEvents(events []domain.ActivityEvent) ([]ActivityItem, error) {
	out := make([]ActivityItem, 0, len(events))
	userCache := map[uint]string{}
	for _, ev := range events {
		actor := userCache[ev.ActorID]
		if actor == "" {
			if u, err := uc.users.FindByID(ev.ActorID); err == nil {
				actor = u.Username
				userCache[ev.ActorID] = actor
			}
		}

		owner := userCache[ev.OwnerID]
		if owner == "" {
			if u, err := uc.users.FindByID(ev.OwnerID); err == nil {
				owner = u.Username
				userCache[ev.OwnerID] = owner
			}
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

	_ = store.Create(ev)
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
