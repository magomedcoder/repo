package usecase

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/metrics"
)

type createMirrorSpec struct {
	Direction string
	URL       string
	Username  string
	Password  string
	SSHKey    string
	SSHPass   string
	Interval  string
	Refspecs  string
	MaxBytes  int64
	GroupID   uint
}

func (uc *RepositoryUseCase) prepareCreateMirror(in CreateRepositoryInput) (*createMirrorSpec, error) {
	mirrorURL := strings.TrimSpace(in.MirrorURL)
	if mirrorURL == "" {
		if strings.TrimSpace(in.MirrorDirection) != "" {
			return nil, ErrInvalidMirrorURL
		}

		return nil, nil
	}

	if err := uc.validateMirrorURL(mirrorURL); err != nil {
		return nil, err
	}

	dir, err := normalizeMirrorDirection(in.MirrorDirection)
	if err != nil {
		return nil, err
	}

	if dir == domain.MirrorNone {
		dir = domain.MirrorPull
	}

	return uc.sealCreateMirror(&createMirrorSpec{
		Direction: dir,
		URL:       mirrorURL,
		Username:  strings.TrimSpace(in.MirrorUsername),
		Password:  in.MirrorPassword,
		SSHKey:    in.MirrorSSHPrivateKey,
		SSHPass:   in.MirrorSSHPassphrase,
		Interval:  in.MirrorInterval,
		Refspecs:  in.MirrorRefspecs,
		MaxBytes:  in.MirrorMaxBytes,
		GroupID:   in.MirrorGroupID,
	})
}

func (uc *RepositoryUseCase) sealCreateMirror(m *createMirrorSpec) (*createMirrorSpec, error) {
	var err error

	if m.MaxBytes < 0 {
		return nil, ErrInvalidMirrorOption
	}

	if m.Interval, err = normalizeInterval(m.Interval); err != nil {
		return nil, err
	}

	if m.Refspecs, err = normalizeRefspecs(m.Refspecs); err != nil {
		return nil, err
	}

	if isSSHMirrorURL(m.URL) && strings.TrimSpace(m.SSHKey) == "" {
		return nil, ErrInvalidSSHKey
	}

	if err := validateSSHKey(m.SSHKey, m.SSHPass); err != nil {
		return nil, err
	}

	if m.Password, err = uc.sealSecret(m.Password); err != nil {
		return nil, err
	}

	if m.SSHKey, err = uc.sealSecret(m.SSHKey); err != nil {
		return nil, err
	}

	if m.SSHPass, err = uc.sealSecret(m.SSHPass); err != nil {
		return nil, err
	}

	return m, nil
}

func applyCreateMirror(repo *domain.Repository, m *createMirrorSpec) {
	if m == nil {
		return
	}

	repo.MirrorDirection = m.Direction
	repo.MirrorURL = m.URL
	repo.MirrorUsername = m.Username
	repo.MirrorPassword = m.Password
	repo.MirrorSSHPrivateKey = m.SSHKey
	repo.MirrorSSHPassphrase = m.SSHPass
	repo.MirrorInterval = m.Interval
	repo.MirrorRefspecs = m.Refspecs
	repo.MirrorMaxBytes = m.MaxBytes
	repo.MirrorGroupID = m.GroupID
	repo.MirrorEnabled = true
	repo.MirrorForce = true
	repo.MirrorPrune = false
	repo.MirrorConflictPolicy = domain.MirrorConflictForce
	repo.MirrorStatus = domain.MirrorStatusIdle
}

func (uc *RepositoryUseCase) cloneNewMirror(ctx context.Context, repo *domain.Repository, actorID uint) error {
	started := time.Now().UTC()

	opts, err := uc.mirrorOptions(repo)
	if err != nil {
		uc.cleanupFailedClone(repo, actorID, err)

		return err
	}

	cctx, cancel := context.WithTimeout(ctx, uc.mirrorConfig().CloneTimeout)
	defer cancel()

	pc := &progressCapture{}
	opts.Progress = pc

	detected, err := uc.git.MirrorClone(cctx, repo.Path, repo.MirrorURL, opts)
	if err != nil {
		uc.recordCreateSync(repo, started, actorID, err)
		uc.cleanupFailedClone(repo, actorID, err)

		return ErrMirrorFailed
	}

	if detected != "" && validateBranchName(detected) == nil {
		repo.DefaultBranch = detected
	}

	_ = uc.afterPull(ctx, repo, refSnapshot{}, pc)
	now := time.Now().UTC()
	repo.MirrorLastSyncAt = &now
	repo.MirrorStatus = domain.MirrorStatusOK

	if iv := effectiveMirrorInterval(repo, uc.mirrorConfig().PullInterval); iv > 0 {
		next := now.Add(iv)
		repo.MirrorNextSyncAt = &next
	}

	if err := uc.store.UpdateMirrorState(ctx, repo); err != nil {
		slog.Warn("mirror state", "repo_id", repo.ID, "err", err)
	}

	uc.recordCreateSync(repo, started, actorID, nil)

	return nil
}

func (uc *RepositoryUseCase) recordCreateSync(repo *domain.Repository, started time.Time, actorID uint, syncErr error) {
	now := time.Now().UTC()
	ev := &domain.MirrorSyncEvent{
		RepoID:     repo.ID,
		Direction:  domain.MirrorPull,
		Trigger:    domain.MirrorTriggerCreate,
		Status:     domain.MirrorEventOK,
		StartedAt:  started,
		FinishedAt: &now,
		DurationMs: now.Sub(started).Milliseconds(),
		ActorID:    actorID,
	}

	result := metrics.ResultOK
	if syncErr != nil {
		ev.Status = domain.MirrorEventFailed
		ev.Error = truncateErr(uc.sanitizeMirrorErr(repo, repo.MirrorURL, syncErr), 500)
		result = metrics.ResultFail
	}

	ctx := context.Background()
	uc.createEvent(ctx, ev)
	uc.mirror.metrics.ObserveSync(domain.MirrorPull, result, now.Sub(started))
	uc.logSync(repo, ev, classifyMirrorErr(syncErr), ev.Error)
}

func (uc *RepositoryUseCase) cleanupFailedClone(repo *domain.Repository, actorID uint, cause error) {
	if uc.mirror.events != nil {
		if err := uc.mirror.events.DeleteByRepo(context.Background(), repo.ID); err != nil {
			slog.Warn("mirror cleanup", "repo_id", repo.ID, "err", err)
		}
	}

	if err := uc.git.Remove(repo.Path); err != nil {
		slog.Error("mirror cleanup", "repo_id", repo.ID, "err", err)
	}

	if err := uc.store.Delete(repo.ID); err != nil {
		slog.Error("mirror cleanup", "repo_id", repo.ID, "err", err)
	}

	slog.Warn("mirror create failed", "repo_id", repo.ID, "err", truncateErr(uc.sanitizeMirrorErr(repo, repo.MirrorURL, cause), 300))

	recordActivity(uc.activity, &domain.ActivityEvent{
		ActorID: actorID,
		OwnerID: repo.OwnerID,
		Type:    domain.ActivityMirrorSync,
		Summary: activitySummary("pull mirror %s failed to clone and was removed", repo.Name),
	})
}
