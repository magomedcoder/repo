package usecase

import (
	"context"
	"strings"

	"magomedcoder.ru/repo/internal/domain"
)

func ResolveReadable(repos domain.RepositoryStore, authz *Authz, in ResolveRepositoryInput) (*domain.Repository, error) {
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

	repo, err := repos.FindByRepoPath(context.Background(), logical)
	if err != nil {
		return nil, ErrRepoNotFound
	}

	if authz == nil {
		authz = &Authz{}
	}

	if err := authz.CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	return repo, nil
}
