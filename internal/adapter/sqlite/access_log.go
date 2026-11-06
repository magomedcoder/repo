package sqlite

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type AccessLog struct {
	gormutil.Repo[model.AccessLog]
}

func NewAccessLogRepo(db *gorm.DB) *AccessLog {
	return &AccessLog{Repo: gormutil.NewRepo[model.AccessLog](db)}
}

func (s *AccessLog) Create(log *domain.AccessLog) error {
	m := &model.AccessLog{
		Action:    log.Action,
		IP:        log.IP,
		UserAgent: log.UserAgent,
		UserID:    log.UserID,
		Username:  log.Username,
		Path:      log.Path,
		Repo:      log.Repo,
		Ref:       log.Ref,
	}
	if err := s.Repo.Create(context.Background(), m); err != nil {
		return err
	}

	log.ID = m.ID
	log.CreatedAt = m.CreatedAt

	return nil
}

func (s *AccessLog) List(filter domain.AccessLogFilter) ([]domain.AccessLog, int, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	offset := max(filter.Offset, 0)
	ctx := context.Background()

	q := s.Model(ctx)
	if action := strings.TrimSpace(filter.Action); action != "" {
		q = q.Where("action = ?", action)
	}

	if ip := strings.TrimSpace(filter.IP); ip != "" {
		like := "%" + escapeLike(ip) + "%"
		q = q.Where(`ip LIKE ? ESCAPE '\'`, like)
	}

	if username := strings.TrimSpace(filter.Username); username != "" {
		like := "%" + escapeLike(username) + "%"
		q = q.Where(`username LIKE ? ESCAPE '\'`, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []model.AccessLog
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.AccessLog, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, int(total), nil
}

func (s *AccessLog) DeleteOlderThan(before time.Time) error {
	return s.DeleteWhere(context.Background(), "created_at < ?", before)
}
