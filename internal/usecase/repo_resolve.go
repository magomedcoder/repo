package usecase

import (
	"context"

	"magomedcoder.ru/repo/internal/domain"
)

func ResolveReadable(ctx context.Context, repos domain.RepositoryStore, authz *Authz, in ResolveRepositoryInput) (*domain.Repository, error) {
	name, err := normalizeRepoName(in.Name)
	if err != nil {
		return nil, err
	}

	logical := name + ".git"

	if _, err := domain.ParseRepoPath(logical); err != nil {
		return nil, ErrRepoNotFound
	}

	if authz == nil {
		authz = &Authz{}
	}

	repo, err := repos.FindByRepoPath(ctx, logical)
	if err != nil {
		if in.ViewerID == 0 && authz.AnonymousReadDenied() {
			return nil, ErrUnauthorized
		}

		return nil, ErrRepoNotFound
	}

	if err := authz.CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	return repo, nil
}
