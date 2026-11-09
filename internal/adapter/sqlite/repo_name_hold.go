package sqlite

import (
	"context"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RepoNameHold struct {
	gormutil.Repo[model.RepoNameHold]
}

func NewRepoNameHoldRepo(db *gorm.DB) *RepoNameHold {
	return &RepoNameHold{Repo: gormutil.NewRepo[model.RepoNameHold](db)}
}

func (s *RepoNameHold) Hold(ctx context.Context, name string, expiresAt time.Time) error {
	m := &model.RepoNameHold{
		Name:      strings.ToLower(name),
		ExpiresAt: expiresAt.UTC(),
	}

	return s.DB(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{
				Name: "name",
			},
		},
		DoUpdates: clause.AssignmentColumns([]string{"expires_at"}),
	}).Create(m).Error
}

func (s *RepoNameHold) IsHeld(ctx context.Context, name string, now time.Time) (bool, error) {
	return s.QueryExist(ctx, "name = ? AND expires_at > ?", strings.ToLower(name), now.UTC())
}

func (s *RepoNameHold) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return s.DeleteWhereRows(ctx, "expires_at <= ?", now.UTC())
}
