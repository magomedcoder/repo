package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"magomedcoder.ru/repo/internal/domain"
)

const (
	maxGrepQueryLen  = 200
	maxGrepLineLen   = 240
	maxGrepFileBytes = 1 << 20
)

func (r *Repository) Grep(ctx context.Context, repoPath, query string, limit int) ([]domain.CodeSearchHit, error) {
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	if query == "" || len(query) > maxGrepQueryLen {
		return nil, nil
	}

	if limit <= 0 {
		limit = 5
	}

	if limit > 50 {
		limit = 50
	}

	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, nil
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("grep head: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("grep tree: %w", err)
	}

	hits := make([]domain.CodeSearchHit, 0, limit)
	err = tree.Files().ForEach(func(file *object.File) error {
		if err := checkCtx(ctx); err != nil {
			return err
		}

		if len(hits) >= limit {
			return storer.ErrStop
		}

		if file == nil || file.Size <= 0 || file.Size > maxGrepFileBytes {
			return nil
		}

		bin, berr := file.IsBinary()
		if berr != nil || bin {
			return nil
		}

		reader, rerr := file.Reader()
		if rerr != nil {
			return nil
		}

		content, rerr := io.ReadAll(io.LimitReader(reader, maxGrepFileBytes+1))
		_ = reader.Close()
		if rerr != nil || len(content) > maxGrepFileBytes {
			return nil
		}

		sc := bufio.NewScanner(strings.NewReader(string(content)))
		sc.Buffer(make([]byte, 0, 64*1024), maxGrepFileBytes)
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := sc.Text()
			if !strings.Contains(line, query) {
				continue
			}

			hits = append(hits, domain.CodeSearchHit{
				Path:    file.Name,
				Line:    lineNo,
				Content: sanitizeGrepContent(line),
			})
			if len(hits) >= limit {
				return storer.ErrStop
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("grep: %w", err)
	}

	return hits, nil
}

func sanitizeGrepContent(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "\uFFFD")
	}

	s = strings.ReplaceAll(s, "\t", " ")
	if len(s) > maxGrepLineLen {
		s = s[:maxGrepLineLen] + "..."
	}

	return s
}
