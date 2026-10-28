package usecase

import "repo.magomedcoder.ru/repo/internal/domain"

func (uc *RepositoryUseCase) ResolveRepository(in ResolveRepositoryInput) (*domain.Repository, string, string, error) {
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
