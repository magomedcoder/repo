package usecase

import (
	"context"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type AccessLogItem struct {
	ID        uint      `json:"id"`
	Action    string    `json:"action"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	UserID    *uint     `json:"user_id,omitzero"`
	Username  string    `json:"username"`
	Path      string    `json:"path"`
	Repo      string    `json:"repo"`
	Ref       string    `json:"ref"`
	CreatedAt time.Time `json:"created_at"`
}

type AccessLogListResult struct {
	Items  []AccessLogItem `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type AccessLogUseCase struct {
	logs  domain.AccessLogStore
	users domain.UserStore
}

func NewAccessLogUseCase(logs domain.AccessLogStore, users domain.UserStore) *AccessLogUseCase {
	return &AccessLogUseCase{
		logs:  logs,
		users: users,
	}
}

func (uc *AccessLogUseCase) ListFilter(ctx context.Context, actorID uint, filter domain.AccessLogFilter) (*AccessLogListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if actorID == 0 || uc.users == nil {
		return nil, ErrUnauthorized
	}

	actor, err := uc.users.FindByID(ctx, actorID)
	if err != nil || actor == nil {
		return nil, ErrUnauthorized
	}

	if !actor.IsAdmin() {
		return nil, ErrAdminRequired
	}

	offset, limit := ClampPagination(filter.Offset, filter.Limit, DefaultPageLimit, MaxAccessLimit)
	filter.Limit = limit
	filter.Offset = offset

	items, total, err := uc.logs.List(filter)
	if err != nil {
		return nil, err
	}

	out := make([]AccessLogItem, 0, len(items))
	for _, item := range items {
		out = append(out, AccessLogItem{
			ID:        item.ID,
			Action:    item.Action,
			IP:        item.IP,
			UserAgent: item.UserAgent,
			UserID:    item.UserID,
			Username:  item.Username,
			Path:      item.Path,
			Repo:      item.Repo,
			Ref:       item.Ref,
			CreatedAt: item.CreatedAt,
		})
	}

	return &AccessLogListResult{
		Items:  out,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
