package usecase

import (
	"context"
	"magomedcoder.ru/repo/internal/domain"
)

func (uc *RepositoryUseCase) ResolveRepository(ctx context.Context, in ResolveRepositoryInput) (*domain.Repository, string, string, error) {
	_ = ctx.Err()
	return uc.resolve(in)
}

func ResolveReadable(repos domain.RepositoryStore, folders domain.FolderStore, users domain.UserStore, in ResolveRepositoryInput) (*domain.Repository, error) {
	uc := &RepositoryUseCase{store: repos, folders: folders, users: users}
	repo, _, _, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if err := CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	return repo, nil
}
