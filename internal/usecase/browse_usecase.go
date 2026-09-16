package usecase

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrRefNotFound  = errors.New("ref not found")
	ErrPathNotFound = errors.New("path not found")
	ErrEmptyRepo    = errors.New("empty repository")
)

func (uc *RepositoryUseCase) ListBranches(in ResolveRepositoryInput) ([]domain.RefInfo, error) {
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

func (uc *RepositoryUseCase) ListTags(in ResolveRepositoryInput) ([]domain.RefInfo, error) {
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

func (uc *RepositoryUseCase) ListCommits(in ResolveRepositoryInput, ref string, offset, limit int) ([]domain.CommitInfo, error) {
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

func (uc *RepositoryUseCase) GetCommit(in ResolveRepositoryInput, sha string) (*domain.CommitInfo, error) {
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

func (uc *RepositoryUseCase) ListTree(in ResolveRepositoryInput, ref, path string) ([]domain.TreeEntry, error) {
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

func (uc *RepositoryUseCase) GetBlob(in ResolveRepositoryInput, ref, path string) (*domain.BlobContent, error) {
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

func (uc *RepositoryUseCase) GetReadme(in ResolveRepositoryInput, ref string) (*domain.BlobContent, error) {
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

func (uc *RepositoryUseCase) GetCommitDiff(in ResolveRepositoryInput, sha string) (*domain.CommitDiff, error) {
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

	return diff, nil
}

func (uc *RepositoryUseCase) GetStats(in ResolveRepositoryInput, ref string) (*domain.RepoStats, error) {
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

	if err := uc.authorizeRead(repo, in.ViewerID); err != nil {
		return nil, err
	}

	return repo, nil
}

func mapGitBrowseErr(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "empty repository"):
		return ErrEmptyRepo
	case strings.Contains(msg, "ref not found"),
		strings.Contains(msg, "reference not found"):
		return ErrRefNotFound
	case strings.Contains(msg, "commit not found"),
		strings.Contains(msg, "file not found"),
		strings.Contains(msg, "path not found"),
		strings.Contains(msg, "readme not found"),
		strings.Contains(msg, "path is not a directory"):
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
	return offset, limit
}
