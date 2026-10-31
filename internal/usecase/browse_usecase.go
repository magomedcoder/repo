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
)

func (uc *RepositoryUseCase) ListBranches(ctx context.Context, in ResolveRepositoryInput) ([]domain.RefInfo, error) {
	_ = ctx.Err()
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
	_ = ctx.Err()
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
	_ = ctx.Err()
	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	if limit <= 0 {
		limit = 30
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

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
	_ = ctx.Err()
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
	_ = ctx.Err()
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
	_ = ctx.Err()
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
	_ = ctx.Err()
	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(ref) == "" {
		ref = repo.DefaultBranch
	}

	blame, err := uc.git.GetBlame(repo.Path, ref, path)
	if err != nil {
		return nil, mapGitBrowseErr(err)
	}

	return blame, nil
}

func (uc *RepositoryUseCase) GetReadme(ctx context.Context, in ResolveRepositoryInput, ref string) (*domain.BlobContent, error) {
	_ = ctx.Err()
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
	_ = ctx.Err()
	repo, err := uc.resolveReadable(in)
	if err != nil {
		return nil, err
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		return nil, ErrPathNotFound
	}

	diff, err := uc.git.GetCommitDiff(repo.Path, sha)
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
	_ = ctx.Err()
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

	if err := CanReadRepo(repo, in.ViewerID); err != nil {
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

func NormalizePagination(offset, limit int) (int, int) {
	if limit < 0 {
		limit = 0
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
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
	_ = ctx.Err()
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
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		var e error
		branches, e = uc.git.ListBranches(repo.Path)
		if e != nil {
			branches = []domain.RefInfo{}
		}
		return nil
	})
	g.Go(func() error {
		var e error
		tree, e = uc.git.ListTree(repo.Path, ref, "")
		if e != nil {
			tree = []domain.TreeEntry{}
		}
		return nil
	})
	g.Go(func() error {
		stats, _ = uc.git.GetStats(repo.Path, ref)
		return nil
	})
	g.Go(func() error {
		readme, _ = uc.git.FindReadme(repo.Path, ref)
		return nil
	})
	_ = g.Wait()

	return &RepositorySummary{
		Branches: branches,
		Tree:     tree,
		Stats:    stats,
		Readme:   readme,
	}, nil
}
