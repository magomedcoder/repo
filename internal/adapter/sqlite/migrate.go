package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	sqlmig "git.magomedcoder.ru/repo/migrations/sqlite"
	"gorm.io/gorm"
)

type migration struct {
	version int
	name    string
	sql     string
}

func applyMigrations(db *gorm.DB, file string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("sql db: %w", err)
	}

	ctx := context.Background()
	if err := integrityCheck(ctx, sqlDB); err != nil {
		return err
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("sql conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	pending, err := pendingMigrations(ctx, conn)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return nil
	}

	if err := backupDatabase(ctx, sqlDB, file); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(ctx, "ROLLBACK")
		}
	}()

	pending, err = pendingMigrations(ctx, conn)
	if err != nil {
		return err
	}
	for _, m := range pending {
		if err := execScript(ctx, conn, m.sql); err != nil {
			return fmt.Errorf("migration %d %s: %w", m.version, m.name, err)
		}

		if _, err := conn.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`, m.version, m.name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}

	committed = true

	if err := integrityCheck(ctx, sqlDB); err != nil {
		return err
	}

	return nil
}

func pendingMigrations(ctx context.Context, conn *sql.Conn) ([]migration, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := map[int]bool{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}

		applied[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	all, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	var pending []migration
	for _, m := range all {
		if !applied[m.version] {
			pending = append(pending, m)
		}
	}

	return pending, nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(sqlmig.Files, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		names = append(names, entry.Name())
	}
	sort.Strings(names)

	out := make([]migration, 0, len(names))
	seen := map[int]string{}
	for _, name := range names {
		version, title, err := parseMigrationName(name)
		if err != nil {
			return nil, err
		}

		if prev, ok := seen[version]; ok {
			return nil, fmt.Errorf("duplicate migration version %d: %s and %s", version, prev, name)
		}

		seen[version] = name
		body, err := sqlmig.Files.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}

		out = append(out, migration{
			version: version,
			name:    title,
			sql:     string(body),
		})
	}

	return out, nil
}

func parseMigrationName(name string) (int, string, error) {
	base := strings.TrimSuffix(name, ".up.sql")
	num, title, ok := strings.Cut(base, "_")
	if !ok || title == "" {
		return 0, "", fmt.Errorf("migration file %s must look like 000_migration.up.sql", name)
	}

	version, err := strconv.Atoi(num)
	if err != nil || version < 0 {
		return 0, "", fmt.Errorf("migration file %s has no version", name)
	}

	return version, title, nil
}

func execScript(ctx context.Context, conn *sql.Conn, script string) error {
	for _, stmt := range splitSQL(script) {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return err
		}
	}

	return nil
}

func splitSQL(script string) []string {
	var out []string
	var b strings.Builder
	inSingle := false
	for i := 0; i < len(script); i++ {
		ch := script[i]
		if ch == '\'' {
			b.WriteByte(ch)
			if inSingle && i+1 < len(script) && script[i+1] == '\'' {
				b.WriteByte('\'')
				i++
				continue
			}

			inSingle = !inSingle
			continue
		}

		if ch == ';' && !inSingle {
			stmt := strings.TrimSpace(b.String())
			if stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
			continue
		}

		b.WriteByte(ch)
	}
	if stmt := strings.TrimSpace(b.String()); stmt != "" {
		out = append(out, stmt)
	}

	return out
}

func backupDatabase(ctx context.Context, sqlDB *sql.DB, file string) error {
	if file == "" || file == ":memory:" || strings.HasPrefix(file, "file::memory:") {
		return nil
	}

	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	dest := file + "." + stamp + ".bak"
	escaped := strings.ReplaceAll(dest, "'", "''")
	if _, err := sqlDB.ExecContext(ctx, `VACUUM INTO '`+escaped+`'`); err != nil {
		return fmt.Errorf("backup database: %w", err)
	}

	if err := os.Chmod(dest, 0o600); err != nil {
		return fmt.Errorf("backup mode: %w", err)
	}

	return nil
}

func integrityCheck(ctx context.Context, sqlDB *sql.DB) error {
	rows, err := sqlDB.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return fmt.Errorf("integrity_check: %w", err)
	}
	defer rows.Close()

	var problems []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return fmt.Errorf("integrity_check: %w", err)
		}

		if line != "ok" {
			problems = append(problems, line)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("integrity_check: %w", err)
	}

	if len(problems) > 0 {
		return fmt.Errorf("integrity_check: %s", strings.Join(problems, "; "))
	}

	return nil
}
