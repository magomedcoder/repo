package usecase

import (
	"context"
	"strings"

	"magomedcoder.ru/repo/internal/domain"
)

func (uc *RepositoryUseCase) ResolveRepository(ctx context.Context, in ResolveRepositoryInput) (*domain.Repository, string, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", "", err
	}
	return uc.resolve(in)
}

func ResolveReadable(repos domain.RepositoryStore, _ domain.FolderStore, _ domain.UserStore, in ResolveRepositoryInput) (*domain.Repository, error) {
	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, err
	}

	folderPath := strings.Trim(strings.TrimSpace(in.FolderPath), "/")
	logical := name + ".git"
	if folderPath != "" {
		logical = folderPath + "/" + name + ".git"
	}

	if _, err := domain.ParseRepoPath(logical); err != nil {
		return nil, ErrRepoNotFound
	}

	repo, err := repos.FindByRepoPath(logical)
	if err != nil {
		return nil, ErrRepoNotFound
	}

	if err := CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	return repo, nil
}
