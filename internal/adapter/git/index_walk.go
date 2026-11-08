package git

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/go-git/go-git/v6/plumbing/storer"
	"magomedcoder.ru/repo/internal/domain"
)

const (
	maxIndexFileBytes  = 1 << 20
	maxIndexFiles      = 5000
	maxIndexTotalBytes = 32 << 20
)

func (r *Repository) TipSHA(repoPath string) (string, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return "", err
	}

	ref, err := repo.Head()
	if err != nil {
		return "", nil
	}

	return ref.Hash().String(), nil
}

func (r *Repository) WalkTextFiles(ctx context.Context, repoPath string) (tipSHA string, files []domain.IndexedTextFile, err error) {
	if err := checkCtx(ctx); err != nil {
		return "", nil, err
	}

	repo, err := r.open(repoPath)
	if err != nil {
		return "", nil, err
	}

	ref, err := repo.Head()
	if err != nil {
		return "", nil, nil
	}
	tipSHA = ref.Hash().String()

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", nil, fmt.Errorf("index tip commit: %w", err)
	}

	if commit.Hash == plumbing.ZeroHash {
		return tipSHA, nil, nil
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", nil, fmt.Errorf("index tip tree: %w", err)
	}

	files = make([]domain.IndexedTextFile, 0, 64)
	var total int
	err = tree.Files().ForEach(func(file *object.File) error {
		if err := checkCtx(ctx); err != nil {
			return err
		}

		if len(files) >= maxIndexFiles || total >= maxIndexTotalBytes {
			return storer.ErrStop
		}

		if file == nil || file.Size <= 0 || file.Size > maxIndexFileBytes {
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

		raw, rerr := io.ReadAll(io.LimitReader(reader, maxIndexFileBytes+1))
		_ = reader.Close()
		if rerr != nil || len(raw) > maxIndexFileBytes {
			return nil
		}

		if !utf8.Valid(raw) {
			raw = []byte(strings.ToValidUTF8(string(raw), "\uFFFD"))
		}

		files = append(files, domain.IndexedTextFile{
			Path: file.Name,
			Body: string(raw),
		})
		total += len(raw)
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return tipSHA, nil, fmt.Errorf("index walk: %w", err)
	}

	return tipSHA, files, nil
}
