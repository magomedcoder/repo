package model

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type Label struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepositoryID uint   `gorm:"not null;uniqueIndex:idx_label_name"`
	Name         string `gorm:"size:50;not null;uniqueIndex:idx_label_name"`
	Color        string `gorm:"size:16;not null"`
}

func (Label) TableName() string {
	return "labels"
}

func LabelFromDomain(l *domain.Label) *Label {
	return &Label{
		ID:           l.ID,
		RepositoryID: l.RepositoryID,
		Name:         l.Name,
		Color:        l.Color,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

func (m Label) ToDomain() domain.Label {
	return domain.Label{
		ID:           m.ID,
		RepositoryID: m.RepositoryID,
		Name:         m.Name,
		Color:        m.Color,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
