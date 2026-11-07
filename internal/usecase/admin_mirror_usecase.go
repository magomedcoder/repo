package usecase

import (
	"context"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/mirror/metrics"
)

type AdminMirrorUseCase struct {
	repos *RepositoryUseCase
}

func NewAdminMirrorUseCase(repos *RepositoryUseCase) *AdminMirrorUseCase {
	return &AdminMirrorUseCase{repos: repos}
}

func (uc *AdminMirrorUseCase) requireActor(ctx context.Context, actorID uint) error {
	return uc.repos.requireSiteAdmin(ctx, actorID)
}

func (uc *AdminMirrorUseCase) List(ctx context.Context, actorID uint, filter domain.MirrorListFilter) ([]AdminMirrorItem, error) {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return nil, err
	}

	return uc.repos.ListAdminMirrors(ctx, actorID, filter)
}

func (uc *AdminMirrorUseCase) Verify(ctx context.Context, actorID uint, filter domain.MirrorListFilter) ([]AdminMirrorVerifyItem, error) {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return nil, err
	}

	return uc.repos.AdminVerifyMirrors(ctx, actorID, filter)
}

func (uc *AdminMirrorUseCase) BulkPause(ctx context.Context, actorID uint, repoIDs []uint, pause bool) (int, error) {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return 0, err
	}

	return uc.repos.BulkPauseMirrors(ctx, actorID, repoIDs, pause)
}

func (uc *AdminMirrorUseCase) BulkSync(ctx context.Context, actorID uint, repoIDs []uint) (int, error) {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return 0, err
	}

	if len(repoIDs) == 0 {
		return uc.repos.BulkSyncAllMirrors(ctx, actorID)
	}

	return uc.repos.BulkSyncMirrors(ctx, actorID, repoIDs)
}

func (uc *AdminMirrorUseCase) Health(ctx context.Context, actorID uint, olderThan time.Duration) (*MirrorHealthItem, error) {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return nil, err
	}

	return uc.repos.MirrorHealth(ctx, olderThan)
}

func (uc *AdminMirrorUseCase) Metrics() *metrics.Metrics {
	return uc.repos.MirrorMetrics()
}

func (uc *AdminMirrorUseCase) ListGroups(ctx context.Context, actorID uint) ([]domain.MirrorGroup, error) {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return nil, err
	}

	return uc.repos.ListMirrorGroups(ctx, actorID)
}

func (uc *AdminMirrorUseCase) CreateGroup(ctx context.Context, actorID uint, g domain.MirrorGroup) (*domain.MirrorGroup, error) {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return nil, err
	}

	return uc.repos.CreateMirrorGroup(ctx, actorID, g)
}

func (uc *AdminMirrorUseCase) DeleteGroup(ctx context.Context, actorID, groupID uint) error {
	if err := uc.requireActor(ctx, actorID); err != nil {
		return err
	}

	return uc.repos.DeleteMirrorGroup(ctx, actorID, groupID)
}

func (uc *AdminMirrorUseCase) ImportOrg(ctx context.Context, in ImportMirrorsInput) ([]ImportMirrorResult, error) {
	if err := uc.requireActor(ctx, in.ViewerID); err != nil {
		return nil, err
	}

	return uc.repos.ImportOrgMirrors(ctx, in)
}
