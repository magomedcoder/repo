package sqlite

import (
	"github.com/magomedcoder/repo/pkg/sqliteutil"
	"gorm.io/gorm"
)

func CheckpointWAL(db *gorm.DB) error {
	return sqliteutil.CheckpointWAL(db)
}
