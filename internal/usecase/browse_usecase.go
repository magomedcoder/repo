package usecase

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
	gitadapter "magomedcoder.ru/repo/internal/adapter/git"
	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrRefNotFound  = NewError(KindNotFound, "ref_not_found")
	ErrPathNotFound = NewError(KindNotFound, "path_not_found")
	ErrEmptyRepo    = NewError(KindNotFound, "empty_repository")

	ErrBlameTooLarge   = NewError(KindValidation, "blame_too_large")
	ErrHistoryTooLarge = NewError(KindValidation, "history_too_large")
)

func (uc *RepositoryUseCase) ListBranches(ctx context.Context, in ResolveRepositoryInput) ([]domain.RefInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	refs, err := uc.git.ListBranches(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	if refs == nil {
		refs = []domain.RefInfo{}
	}

	return refs, nil
}

func (uc *RepositoryUseCase) ListTags(ctx context.Context, in ResolveRepositoryInput) ([]domain.RefInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	refs, err := uc.git.ListTags(repo.Path)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	if refs == nil {
		refs = []domain.RefInfo{}
	}

	return refs, nil
}

func (uc *RepositoryUseCase) ListCommits(ctx context.Context, in ResolveRepositoryInput, ref string, offset, limit int) ([]domain.CommitInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	offset, limit = NormalizePagination(offset, limit)

	commits, err := uc.git.ListCommits(repo.Path, ref, offset, limit)
	if err != nil {
		if isEmptyOrMissingRef(err) {
			return []domain.CommitInfo{}, nil
		}

		return nil, mapGitBrowseErr(err)
	}

	if commits == nil {
		commits = []domain.CommitInfo{}
	}

	return commits, nil
}

func (uc *RepositoryUseCase) GetCommit(ctx context.Context, in ResolveRepositoryInput, sha string) (*domain.CommitInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		return nil, ErrPathNotFound
	}

	commit, err := uc.git.GetCommit(repo.Path, sha)
	if err != nil {
		return nil, mapGitBrowseErr(err)
	}

	return commit, nil
}

func (uc *RepositoryUseCase) ListTree(ctx context.Context, in ResolveRepositoryInput, ref, path string) ([]domain.TreeEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	entries, err := uc.git.ListTree(repo.Path, ref, path)
	if err != nil {
		if isEmptyOrMissingRef(err) {
			return []domain.TreeEntry{}, nil
		}

		return nil, mapGitBrowseErr(err)
	}

	if entries == nil {
		entries = []domain.TreeEntry{}
	}

	return entries, nil
}

func (uc *RepositoryUseCase) GetBlob(ctx context.Context, in ResolveRepositoryInput, ref, path string) (*domain.BlobContent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	blob, err := uc.git.GetBlob(repo.Path, ref, path)
	if err != nil {
		return nil, mapGitBrowseErr(err)
	}

	return blob, nil
}

func (uc *RepositoryUseCase) GetBlame(ctx context.Context, in ResolveRepositoryInput, ref, path string) (*domain.BlameResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	blame, err := uc.git.GetBlame(ctx, repo.Path, ref, path)
	if err != nil {
		return nil, mapGitBrowseErr(err)
	}

	return blame, nil
}

func (uc *RepositoryUseCase) GetReadme(ctx context.Context, in ResolveRepositoryInput, ref string) (*domain.BlobContent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	blob, err := uc.git.FindReadme(repo.Path, ref)
	if err != nil {
		return nil, mapGitBrowseErr(err)
	}

	return blob, nil
}

func (uc *RepositoryUseCase) GetCommitDiff(ctx context.Context, in ResolveRepositoryInput, sha string) (*domain.CommitDiff, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		return nil, ErrPathNotFound
	}

	diff, err := uc.git.GetCommitDiff(ctx, repo.Path, sha)
	if err != nil {
		return nil, mapGitBrowseErr(err)
	}

	const maxPatch = 64 * 1024
	for i := range diff.Files {
		if len(diff.Files[i].Patch) > maxPatch {
			diff.Files[i].Patch = diff.Files[i].Patch[:maxPatch] + "\n... truncated\n"
		}
	}

	return diff, nil
}

func (uc *RepositoryUseCase) GetStats(ctx context.Context, in ResolveRepositoryInput, ref string) (*domain.RepoStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	stats, err := uc.git.GetStats(repo.Path, ref)
	if err != nil {
		return nil, mapGitBrowseErr(err)
	}

	return stats, nil
}

func (uc *RepositoryUseCase) resolveReadable(in ResolveRepositoryInput) (*domain.Repository, error) {
	repo, _, _, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	return repo, nil
}

func mapGitBrowseErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, gitadapter.ErrEmptyRepository):
		return ErrEmptyRepo
	case errors.Is(err, gitadapter.ErrRefNotFound):
		return ErrRefNotFound
	case errors.Is(err, gitadapter.ErrBlameTooLarge):
		return ErrBlameTooLarge
	case errors.Is(err, gitadapter.ErrHistoryTooLarge):
		return ErrHistoryTooLarge
	case errors.Is(err, gitadapter.ErrCommitNotFound),
		errors.Is(err, gitadapter.ErrFileNotFound),
		errors.Is(err, gitadapter.ErrPathNotFound),
		errors.Is(err, gitadapter.ErrReadmeNotFound),
		errors.Is(err, gitadapter.ErrNotDirectory):
		return ErrPathNotFound
	default:
		return err
	}
}

func isEmptyOrMissingRef(err error) bool {
	mapped := mapGitBrowseErr(err)
	return errors.Is(mapped, ErrEmptyRepo) || errors.Is(mapped, ErrRefNotFound)
}

func ParseOffsetLimit(offsetStr, limitStr string) (offset, limit int) {
	offset, _ = strconv.Atoi(offsetStr)
	limit, _ = strconv.Atoi(limitStr)
	if offset < 0 {
		offset = 0
	}
	return offset, limit
}

const (
	DefaultPageLimit = 50
	MaxPageLimit     = 100
	MaxAccessLimit   = 200
	MaxOffset        = 10000
)

func NormalizePagination(offset, limit int) (int, int) {
	return ClampPagination(offset, limit, DefaultPageLimit, MaxPageLimit)
}

func ClampPagination(offset, limit, defLimit, maxLimit int) (int, int) {
	if defLimit <= 0 {
		defLimit = DefaultPageLimit
	}

	if maxLimit <= 0 {
		maxLimit = MaxPageLimit
	}

	if limit <= 0 {
		limit = defLimit
	}

	if limit > maxLimit {
		limit = maxLimit
	}

	if offset < 0 {
		offset = 0
	}

	if offset > MaxOffset {
		offset = MaxOffset
	}

	return offset, limit
}

type RepositorySummary struct {
	Branches []domain.RefInfo    `json:"branches"`
	Tree     []domain.TreeEntry  `json:"tree"`
	Stats    *domain.RepoStats   `json:"stats,omitzero"`
	Readme   *domain.BlobContent `json:"readme,omitzero"`
}

func (uc *RepositoryUseCase) GetSummary(ctx context.Context, in ResolveRepositoryInput, ref string) (*RepositorySummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	var (
		branches []domain.RefInfo
		tree     []domain.TreeEntry
		stats    *domain.RepoStats
		readme   *domain.BlobContent
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := gctx.Err(); err != nil {
			return err
		}

		var e error
		branches, e = uc.git.ListBranches(repo.Path)
		if e != nil {
			if isEmptyOrMissingRef(e) {
				branches = []domain.RefInfo{}
				return nil
			}
			return mapGitBrowseErr(e)
		}

		return nil
	})
	g.Go(func() error {
		if err := gctx.Err(); err != nil {
			return err
		}

		var e error
		tree, e = uc.git.ListTree(repo.Path, ref, "")
		if e != nil {
			if isEmptyOrMissingRef(e) {
				tree = []domain.TreeEntry{}
				return nil
			}
			return mapGitBrowseErr(e)
		}

		return nil
	})
	g.Go(func() error {
		if err := gctx.Err(); err != nil {
			return err
		}

		s, e := uc.git.GetStats(repo.Path, ref)
		if e != nil {
			if isEmptyOrMissingRef(e) {
				return nil
			}
			return mapGitBrowseErr(e)
		}
		stats = s

		return nil
	})
	g.Go(func() error {
		if err := gctx.Err(); err != nil {
			return err
		}

		r, e := uc.git.FindReadme(repo.Path, ref)
		if e != nil && !isEmptyOrMissingRef(e) {
			return mapGitBrowseErr(e)
		}
		readme = r

		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &RepositorySummary{
		Branches: branches,
		Tree:     tree,
		Stats:    stats,
		Readme:   readme,
	}, nil
}
