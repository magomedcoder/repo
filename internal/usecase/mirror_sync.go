package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

func (uc *RepositoryUseCase) SchedulePushMirror(repoID uint) {
	if repoID == 0 {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := uc.syncMirrorByID(ctx, repoID, domain.MirrorPush); err != nil && err != ErrMirrorNotConfigured {
			slog.Warn("mirror push sync", "repo_id", repoID, "err", err)
		}
	}()
}

func (uc *RepositoryUseCase) RunPullMirrorLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		slog.Info("mirror pull scheduler disabled")
		return
	}

	slog.Info("mirror pull scheduler started", "interval", interval.String())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	uc.syncAllPullMirrors(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("mirror pull scheduler stopped")
			return
		case <-ticker.C:
			uc.syncAllPullMirrors(ctx)
		}
	}
}

func (uc *RepositoryUseCase) syncAllPullMirrors(ctx context.Context) {
	repos, err := uc.store.ListByMirrorDirection(ctx, domain.MirrorPull)
	if err != nil {
		slog.Error("list pull mirrors", "err", err)
		return
	}

	for _, repo := range repos {
		if ctx.Err() != nil {
			return
		}
		id := repo.ID
		if err := uc.syncMirrorByID(ctx, id, domain.MirrorPull); err != nil && err != ErrMirrorNotConfigured {
			slog.Warn("mirror pull sync", "repo_id", id, "err", err)
		}
	}
}

func (uc *RepositoryUseCase) syncMirrorByID(ctx context.Context, repoID uint, wantDir string) error {
	lock := uc.mirrorLock(repoID)
	if !lock.TryLock() {
		return nil
	}
	defer lock.Unlock()

	repo, err := uc.store.FindByID(ctx, repoID)
	if err != nil || repo == nil {
		return ErrRepoNotFound
	}

	if repo.MirrorDirection == domain.MirrorNone || repo.MirrorURL == "" {
		return ErrMirrorNotConfigured
	}

	if wantDir != "" && repo.MirrorDirection != wantDir {
		return ErrMirrorNotConfigured
	}

	return uc.runMirrorSync(ctx, repo)
}

func (uc *RepositoryUseCase) runMirrorSync(ctx context.Context, repo *domain.Repository) error {
	auth := mirrorAuthFromRepo(repo)
	var syncErr error
	switch repo.MirrorDirection {
	case domain.MirrorPull:
		syncErr = uc.git.MirrorFetch(ctx, repo.Path, repo.MirrorURL, auth)
		if syncErr == nil {
			_ = uc.git.EnsureDefaultBranch(repo.Path, repo.DefaultBranch)
			if detected, derr := uc.git.DetectHEADBranch(repo.Path); derr == nil && detected != "" {
				if verr := validateBranchName(detected); verr == nil {
					repo.DefaultBranch = detected
				}
			}
		}
	case domain.MirrorPush:
		syncErr = uc.git.MirrorPush(ctx, repo.Path, repo.MirrorURL, auth)
	default:
		return ErrInvalidMirrorDirection
	}

	now := time.Now().UTC()
	repo.MirrorLastSyncAt = &now
	if syncErr != nil {
		repo.MirrorLastError = truncateErr(syncErr.Error(), 500)
		_ = uc.store.Update(repo)
		return fmt.Errorf("%w: %v", ErrMirrorFailed, syncErr)
	}

	repo.MirrorLastError = ""
	if err := uc.store.Update(repo); err != nil {
		return fmt.Errorf("save mirror sync: %w", err)
	}

	return nil
}

func (uc *RepositoryUseCase) mirrorLock(repoID uint) *sync.Mutex {
	uc.mirrorMu.Lock()
	defer uc.mirrorMu.Unlock()
	if uc.mirrorLocks == nil {
		uc.mirrorLocks = make(map[uint]*sync.Mutex)
	}

	if uc.mirrorLocks[repoID] == nil {
		uc.mirrorLocks[repoID] = &sync.Mutex{}
	}

	return uc.mirrorLocks[repoID]
}
