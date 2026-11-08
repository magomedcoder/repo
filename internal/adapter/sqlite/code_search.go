package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type CodeIndex struct {
	db *gorm.DB
}

func NewCodeIndex(db *gorm.DB) *CodeIndex {
	return &CodeIndex{db: db}
}

func (s *CodeIndex) EnsureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS code_index_meta (
			repo_id INTEGER PRIMARY KEY NOT NULL,
			tip_sha TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS code_index_files (
			id INTEGER PRIMARY KEY,
			repo_id INTEGER NOT NULL,
			path TEXT NOT NULL,
			tip_sha TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_code_index_files_repo_path ON code_index_files (repo_id, path)`,
	}

	for _, stmt := range stmts {
		if err := s.db.WithContext(ctx).Exec(stmt).Error; err != nil {
			return fmt.Errorf("code index schema: %w", err)
		}
	}

	return nil
}

func (s *CodeIndex) ReplaceRepoIndex(ctx context.Context, repoID uint, tipSHA string, files []domain.CodeIndexFile) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM code_index_files WHERE repo_id = ?`, repoID).Error; err != nil {
			return fmt.Errorf("clear files: %w", err)
		}

		for i := range files {
			f := &files[i]
			if err := tx.Exec(
				`INSERT INTO code_index_files(repo_id, path, tip_sha, body) VALUES (?, ?, ?, ?)`,
				repoID, f.Path, tipSHA, f.Body,
			).Error; err != nil {
				return fmt.Errorf("insert file: %w", err)
			}
		}

		now := time.Now().Unix()
		if err := tx.Exec(
			`INSERT INTO code_index_meta(repo_id, tip_sha, updated_at) VALUES (?, ?, ?)
			 ON CONFLICT(repo_id) DO UPDATE SET tip_sha = excluded.tip_sha, updated_at = excluded.updated_at`,
			repoID, tipSHA, now,
		).Error; err != nil {
			return fmt.Errorf("upsert meta: %w", err)
		}

		return nil
	})
}

func (s *CodeIndex) DeleteRepoIndex(ctx context.Context, repoID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM code_index_files WHERE repo_id = ?`, repoID).Error; err != nil {
			return err
		}

		return tx.Exec(`DELETE FROM code_index_meta WHERE repo_id = ?`, repoID).Error
	})
}

func (s *CodeIndex) Search(ctx context.Context, viewerID uint, tokens []string, afterRepoID uint, afterPath string, limit int) ([]domain.CodeIndexHit, error) {
	if limit <= 0 {
		limit = 30
	}
	if len(tokens) == 0 {
		return nil, nil
	}

	args := make([]any, 0, 8+len(tokens))
	var b strings.Builder
	b.WriteString(`SELECT f.repo_id, f.path, f.body
FROM code_index_files f
WHERE f.repo_id IN (SELECT id FROM repositories WHERE deleted_at IS NULL`)

	if viewerID == 0 {
		b.WriteString(` AND is_private = 0`)
	} else {
		b.WriteString(` AND (
			is_private = 0
			OR owner_id = ?
			OR id IN (SELECT repo_id FROM repo_collaborators WHERE user_id = ? AND deleted_at IS NULL)
			OR ` + teamGrantAccessSQL + `
		)`)
		args = append(args, viewerID, viewerID, viewerID)
	}
	b.WriteString(`)`)

	for _, tok := range tokens {
		b.WriteString(` AND lower(f.body) LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(strings.ToLower(tok))+"%")
	}

	if afterRepoID > 0 || afterPath != "" {
		b.WriteString(` AND (f.repo_id > ? OR (f.repo_id = ? AND f.path > ?))`)
		args = append(args, afterRepoID, afterRepoID, afterPath)
	}

	b.WriteString(` ORDER BY f.repo_id ASC, f.path ASC LIMIT ?`)
	args = append(args, limit)

	rows, err := s.db.WithContext(ctx).Raw(b.String(), args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("code search: %w", err)
	}
	defer rows.Close()

	out := make([]domain.CodeIndexHit, 0, limit)
	for rows.Next() {
		var hit domain.CodeIndexHit
		if err := rows.Scan(&hit.RepoID, &hit.Path, &hit.Body); err != nil {
			return nil, err
		}
		out = append(out, hit)
	}

	return out, rows.Err()
}

func (s *CodeIndex) Meta(ctx context.Context, repoID uint) (*domain.CodeIndexMeta, error) {
	var m domain.CodeIndexMeta
	err := s.db.WithContext(ctx).Raw(
		`SELECT repo_id, tip_sha, updated_at FROM code_index_meta WHERE repo_id = ?`, repoID,
	).Scan(&m).Error
	if err != nil {
		return nil, err
	}

	if m.RepoID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return &m, nil
}

func (s *CodeIndex) ListStaleOrMissing(ctx context.Context, limit int) ([]uint, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.WithContext(ctx).Raw(`
		SELECT r.id FROM repositories r
		LEFT JOIN code_index_meta m ON m.repo_id = r.id
		WHERE r.deleted_at IS NULL AND m.repo_id IS NULL
		ORDER BY r.id ASC
		LIMIT ?
	`, limit).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}
