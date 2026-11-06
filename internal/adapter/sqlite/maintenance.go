package sqlite

import "gorm.io/gorm"

func CheckpointWAL(db *gorm.DB) error {
	return db.Exec("PRAGMA wal_checkpoint(PASSIVE)").Error
}
