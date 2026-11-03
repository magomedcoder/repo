package usecase

import "magomedcoder.ru/repo/internal/domain"

func CanReadRepo(repo *domain.Repository, viewerID uint) error {
	if repo == nil {
		return ErrRepoNotFound
	}

	if !repo.IsPrivate {
		return nil
	}

	if CanReadPrivateRepo(repo, viewerID) {
		return nil
	}

	return ErrRepoNotFound
}

func CanReadPrivateRepo(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 {
		return false
	}

	kind := repo.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	return kind == domain.OwnerKindUser && viewerID == repo.OwnerID
}

func CanAdminRepo(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 {
		return false
	}

	kind := repo.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	return kind == domain.OwnerKindUser && viewerID == repo.OwnerID
}

func CanWriteGit(repo *domain.Repository, viewerID uint) bool {
	return CanAdminRepo(repo, viewerID)
}
