package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

const defaultActivityLimit = 30

type ActivityItem struct {
	ID        uint      `json:"id"`
	Type      string    `json:"type"`
	Summary   string    `json:"summary"`
	Ref       string    `json:"ref,omitzero"`
	Actor     string    `json:"actor"`
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
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

func (uc *ActivityUseCase) ListOwner(ctx context.Context, ownerSlug string, viewerID uint, limit int) ([]ActivityItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slug := strings.ToLower(strings.TrimSpace(ownerSlug))
	id, err := uc.resolveOwner(slug)
	if err != nil {
		return nil, err
	}

	_, limit = ClampPagination(0, limit, defaultActivityLimit, MaxPageLimit)

	isOwner := viewerID != 0 && viewerID == id
	var events []domain.ActivityEvent
	if isOwner {
		events, err = uc.events.ListByOwner(id, limit)
	} else {
		events, err = uc.events.ListPublicByOwner(id, limit)
	}

	if err != nil {
		return nil, err
	}

	events, err = uc.filterVisible(events, id, viewerID)
	if err != nil {
		return nil, err
	}

	return uc.mapEvents(events)
}

func (uc *ActivityUseCase) filterVisible(events []domain.ActivityEvent, ownerID, viewerID uint) ([]domain.ActivityEvent, error) {
	isOwner := viewerID != 0 && viewerID == ownerID
	if isOwner {
		return events, nil
	}

	ids, err := uc.repos.ListPublicIDsByOwner(context.Background(), ownerID)
	if err != nil {
		return nil, err
	}

	public := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		public[id] = struct{}{}
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

func (uc *ActivityUseCase) resolveOwner(slug string) (uint, error) {
	if u, err := uc.users.FindByUsername(context.Background(), slug); err == nil {
		return u.ID, nil
	}

	return 0, ErrRepoNotFound
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

	ids := slices.Collect(maps.Keys(idSet))

	usersByID, err := uc.users.FindByIDs(context.Background(), ids)
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
		slog.Error("activity: create event", "err", err)
	}
}

func activitySummary(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

func recordRepoActivity(store domain.ActivityStore, repo *domain.Repository, actorID uint, typ, summary string) {
	if repo == nil {
		return
	}

	recordActivity(store, &domain.ActivityEvent{
		ActorID: actorID,
		OwnerID: repo.OwnerID,
		RepoID:  &repo.ID,
		Type:    typ,
		Summary: summary,
	})
}
