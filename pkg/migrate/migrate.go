package migrate

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

type Migration struct {
	Version     int
	Name        string
	Destructive bool
	Up          func(db *gorm.DB) error
	Down        func(db *gorm.DB) error
}

type Record struct {
	Version   int       `gorm:"primaryKey"`
	Name      string    `gorm:"size:128;not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func (Record) TableName() string {
	return "schema_migrations"
}

type Options struct {
	DB         *gorm.DB
	DBPath     string
	Migrations []Migration
	Bootstrap  func(db *gorm.DB) error
	Logger     *slog.Logger
}

func Run(opts Options) error {
	if opts.DB == nil {
		return errors.New("migrate: DB is required")
	}

	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	migrations := opts.Migrations

	empty, err := IsEmptySQLite(opts.DB)
	if err != nil {
		return err
	}

	if err := ensureJournal(opts.DB); err != nil {
		return err
	}

	if empty {
		log.Info("migrate: fresh database, bootstrapping")
		if opts.Bootstrap != nil {
			if err := opts.Bootstrap(opts.DB); err != nil {
				return err
			}
		}

		if err := markAllApplied(opts.DB, migrations); err != nil {
			return err
		}
		log.Info("migrate: bootstrap complete", "migrations", len(migrations))
		return nil
	}

	applied, err := Applied(opts.DB)
	if err != nil {
		return err
	}

	pending := Pending(migrations, applied)
	log.Info("migrate: status", "applied", len(applied), "pending", len(pending), "latest", LatestVersion(migrations))

	for _, m := range pending {
		log.Info("migrate: pending", "version", m.Version, "name", m.Name)
	}

	if len(pending) == 0 {
		return nil
	}

	if NeedsBackup(pending) {
		path, err := BackupSQLite(opts.DBPath)
		if err != nil {
			return fmt.Errorf("backup before destructive migrate: %w", err)
		}

		if path != "" {
			log.Info("migrate: backup created", "path", path)
		}
	}

	for _, m := range pending {
		log.Info("migrate: applying", "version", m.Version, "name", m.Name)
		if m.Up == nil {
			return fmt.Errorf("migrate v%d %s: Up is nil", m.Version, m.Name)
		}

		if err := m.Up(opts.DB); err != nil {
			return fmt.Errorf("migrate v%d %s: %w", m.Version, m.Name, err)
		}

		if err := RecordApplied(opts.DB, m); err != nil {
			return fmt.Errorf("journal v%d %s: %w", m.Version, m.Name, err)
		}

		log.Info("migrate: applied", "version", m.Version, "name", m.Name)
	}
	return nil
}

func ensureJournal(db *gorm.DB) error {
	if err := db.AutoMigrate(&Record{}); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	return nil
}

func Applied(db *gorm.DB) (map[int]Record, error) {
	var rows []Record
	if err := db.Order("version ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make(map[int]Record, len(rows))
	for _, row := range rows {
		out[row.Version] = row
	}

	return out, nil
}

func Pending(migrations []Migration, applied map[int]Record) []Migration {
	var out []Migration
	for _, m := range migrations {
		if _, ok := applied[m.Version]; !ok {
			out = append(out, m)
		}
	}

	return out
}

func LatestVersion(migrations []Migration) int {
	latest := 0
	for _, m := range migrations {
		if m.Version > latest {
			latest = m.Version
		}
	}

	return latest
}

func NeedsBackup(pending []Migration) bool {
	for _, m := range pending {
		if m.Destructive {
			return true
		}
	}

	return false
}

func Names(migrations []Migration) []string {
	out := make([]string, 0, len(migrations))
	for _, m := range migrations {
		out = append(out, fmt.Sprintf("%04d_%s", m.Version, m.Name))
	}

	return out
}

func markAllApplied(db *gorm.DB, migrations []Migration) error {
	now := time.Now().UTC()
	for _, m := range migrations {
		row := Record{
			Version:   m.Version,
			Name:      m.Name,
			AppliedAt: now,
		}

		if err := db.Where("version = ?", m.Version).Assign(row).FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}

	return nil
}

func RecordApplied(db *gorm.DB, m Migration) error {
	row := Record{
		Version:   m.Version,
		Name:      m.Name,
		AppliedAt: time.Now().UTC(),
	}

	return db.Create(&row).Error
}
