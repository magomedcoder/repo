package sqlite

import (
	"gorm.io/gorm"
	"magomedcoder.ru/repo/pkg/sqliteutil"
)

func CheckpointWAL(db *gorm.DB) error {
	return sqliteutil.CheckpointWAL(db)
}
