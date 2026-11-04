package sqlite

import (
	"strings"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type accessLogModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Action    string `gorm:"size:32;not null;index"`
	IP        string `gorm:"size:64;not null;default:''"`
	UserAgent string `gorm:"size:512;not null;default:''"`
	UserID    *uint
	Username  string `gorm:"size:64;not null;default:''"`
	Path      string `gorm:"size:1024;not null;default:''"`
	Repo      string `gorm:"size:512;not null;default:''"`
	Ref       string `gorm:"size:255;not null;default:''"`
}

func (accessLogModel) TableName() string { return "access_logs" }

type AccessLogStore struct {
	db *gorm.DB
}

func NewAccessLogStore(db *gorm.DB) *AccessLogStore {
	return &AccessLogStore{db: db}
}

func (s *AccessLogStore) Create(log *domain.AccessLog) error {
	m := &accessLogModel{
		Action:    log.Action,
		IP:        log.IP,
		UserAgent: log.UserAgent,
		UserID:    log.UserID,
		Username:  log.Username,
		Path:      log.Path,
		Repo:      log.Repo,
		Ref:       log.Ref,
	}
	if err := s.db.Create(m).Error; err != nil {
		return err
	}

	log.ID = m.ID
	log.CreatedAt = m.CreatedAt

	return nil
}

func (s *AccessLogStore) List(filter domain.AccessLogFilter) ([]domain.AccessLog, int, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	offset := max(filter.Offset, 0)

	q := s.db.Model(&accessLogModel{})
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

	var models []accessLogModel
	if err := q.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.AccessLog, 0, len(models))
	for _, m := range models {
		out = append(out, domain.AccessLog{
			ID:        m.ID,
			Action:    m.Action,
			IP:        m.IP,
			UserAgent: m.UserAgent,
			UserID:    m.UserID,
			Username:  m.Username,
			Path:      m.Path,
			Repo:      m.Repo,
			Ref:       m.Ref,
			CreatedAt: m.CreatedAt,
		})
	}

	return out, int(total), nil
}

func (s *AccessLogStore) DeleteOlderThan(before time.Time) error {
	return s.db.Where("created_at < ?", before).Delete(&accessLogModel{}).Error
}
