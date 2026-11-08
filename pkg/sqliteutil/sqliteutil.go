package sqliteutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(path string) (*gorm.DB, error) {
	dsn := path
	file, _, _ := strings.Cut(dsn, "?")

	if dir := filepath.Dir(file); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("data dir: %w", err)
		}
	}

	if !strings.Contains(dsn, "_pragma") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn += sep + "_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	if file != "" && file != ":memory:" {
		_ = os.Chmod(file, 0o600)
	}

	return db, nil
}

func CheckpointWAL(db *gorm.DB) error {
	return db.Exec("PRAGMA wal_checkpoint(PASSIVE)").Error
}

func IsEmpty(db *gorm.DB) (bool, error) {
	var name string
	err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' LIMIT 1`).Scan(&name).Error
	if err != nil {
		return false, err
	}

	return name == "", nil
}
