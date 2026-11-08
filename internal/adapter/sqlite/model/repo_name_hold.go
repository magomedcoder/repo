package model

import "time"

type RepoNameHold struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Name      string    `gorm:"size:100;not null;uniqueIndex:idx_repo_name_holds_name"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (RepoNameHold) TableName() string {
	return "repo_name_holds"
}
