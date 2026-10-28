package sqlite

import (
	"time"

	"gorm.io/gorm"
	"repo.magomedcoder.ru/repo/internal/domain"
)

type activityModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	ActorID   uint   `gorm:"not null"`
	OwnerKind string `gorm:"size:8;not null"`
	OwnerID   uint   `gorm:"not null"`
	RepoID    *uint
	Type      string `gorm:"size:32;not null"`
	Summary   string `gorm:"size:512;not null"`
	Ref       string `gorm:"size:255"`
}

func (activityModel) TableName() string {
	return "activity_events"
}

func (m activityModel) toDomain() domain.ActivityEvent {
	return domain.ActivityEvent{
		ID:        m.ID,
		ActorID:   m.ActorID,
		OwnerKind: m.OwnerKind,
		OwnerID:   m.OwnerID,
		RepoID:    m.RepoID,
		Type:      m.Type,
		Summary:   m.Summary,
		Ref:       m.Ref,
		CreatedAt: m.CreatedAt,
	}
}

type ActivityStore struct {
	db *gorm.DB
}

func NewActivityStore(db *gorm.DB) *ActivityStore {
	return &ActivityStore{db: db}
}

func (s *ActivityStore) Create(ev *domain.ActivityEvent) error {
	model := &activityModel{
		ActorID:   ev.ActorID,
		OwnerKind: ev.OwnerKind,
		OwnerID:   ev.OwnerID,
		RepoID:    ev.RepoID,
		Type:      ev.Type,
		Summary:   ev.Summary,
		Ref:       ev.Ref,
	}
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*ev = model.toDomain()

	return nil
}

func (s *ActivityStore) ListByOwner(ownerKind string, ownerID uint, limit int) ([]domain.ActivityEvent, error) {
	if limit <= 0 {
		limit = 30
	}

	var models []activityModel
	err := s.db.Where("owner_kind = ? AND owner_id = ?", ownerKind, ownerID).Order("created_at DESC").Limit(limit).Find(&models).Error
	if err != nil {
		return nil, err
	}

	return activityModelsToDomain(models), nil
}

func activityModelsToDomain(models []activityModel) []domain.ActivityEvent {
	out := make([]domain.ActivityEvent, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out
}
