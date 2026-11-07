package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/mirror/metrics"
)

var (
	ErrMirrorSyncSkipped = NewError(KindConflict, "mirror_sync_skipped")
	ErrMirrorPaused      = NewError(KindConflict, "mirror_paused")
	ErrMirrorBusy        = NewError(KindConflict, "mirror_busy")
	ErrMirrorRateLimited = NewError(KindRateLimited, "mirror_sync_rate_limited")
)

type syncRequest struct {
	RepoID  uint
	WantDir string
	Trigger string
	ActorID uint
}

func (uc *RepositoryUseCase) SchedulePushMirror(repoID uint) {
	if repoID == 0 {
		return
	}

	cfg := uc.mirrorConfig()
	m := uc.mirror

	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.debounce[repoID]; ok {
		t.Reset(cfg.PushDebounce)

		return
	}

	m.wg.Add(1)
	m.debounce[repoID] = time.AfterFunc(cfg.PushDebounce, func() {
		defer m.wg.Done()

		m.mu.Lock()
		delete(m.debounce, repoID)
		m.mu.Unlock()

		uc.runDebouncedPush(repoID)
	})
}

func (uc *RepositoryUseCase) runDebouncedPush(repoID uint) {
	cfg := uc.mirrorConfig()
	ctx, cancel := context.WithTimeout(uc.mirrorRootContext(), cfg.PushTimeout+time.Minute)
	defer cancel()

	_, err := uc.syncMirrorByID(ctx, syncRequest{RepoID: repoID, WantDir: domain.MirrorPush, Trigger: domain.MirrorTriggerPush})
	if err != nil && !isBenignMirrorErr(err) {
		slog.Warn("mirror push sync", "repo_id", repoID, "err", err)
	}
}

func isBenignMirrorErr(err error) bool {
	return errors.Is(err, ErrMirrorNotConfigured) || errors.Is(err, ErrMirrorPaused) || errors.Is(err, ErrMirrorSyncSkipped)
}

func (uc *RepositoryUseCase) flushPendingPush(repoID uint) {
	uc.mirror.mu.Lock()
	pending := uc.mirror.pending[repoID]
	delete(uc.mirror.pending, repoID)
	uc.mirror.mu.Unlock()

	if pending {
		uc.SchedulePushMirror(repoID)
	}
}

func (uc *RepositoryUseCase) RunPullMirrorLoop(ctx context.Context, interval time.Duration) {
	uc.mirror.mu.Lock()
	uc.mirror.cfg.PullInterval = interval
	uc.mirror.mu.Unlock()

	tick := time.Minute
	if interval > 0 && interval < tick {
		tick = interval
	}

	if interval <= 0 {
		slog.Info("mirror scheduler: global interval disabled, per-repo intervals still apply")
	}

	slog.Info("mirror scheduler started", "tick", tick.String(), "global_interval", interval.String(), "workers", uc.mirrorConfig().Workers)
	uc.resetStaleSyncing(ctx)

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	uc.syncAllPullMirrors(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("mirror scheduler stopped")

			return
		case <-ticker.C:
			uc.syncAllPullMirrors(ctx)
		}
	}
}

func (uc *RepositoryUseCase) resetStaleSyncing(ctx context.Context) {
	if _, err := uc.store.ResetStaleMirrorSyncing(ctx); err != nil {
		slog.Warn("reset stale mirror syncing", "err", err)
	}
}

func (uc *RepositoryUseCase) listAllMirrors(ctx context.Context, f domain.MirrorListFilter) ([]domain.Repository, error) {
	const page = 500

	var out []domain.Repository

	f.Limit = page
	for {
		chunk, err := uc.store.ListMirrors(ctx, f)
		if err != nil {
			return nil, err
		}

		out = append(out, chunk...)
		if len(chunk) < page {
			return out, nil
		}

		f.Offset += page
	}
}

func mirrorDue(repo *domain.Repository, global time.Duration, now time.Time) bool {
	if repo.MirrorDirection == domain.MirrorNone || repo.MirrorURL == "" || !repo.MirrorEnabled {
		return false
	}

	if repo.MirrorNextSyncAt != nil && now.Before(*repo.MirrorNextSyncAt) {
		return false
	}

	if !inSyncWindow(repo.MirrorSyncWindow, now) {
		return false
	}

	iv := effectiveMirrorInterval(repo, global)
	if repo.MirrorDirection == domain.MirrorPull {
		return iv > 0
	}

	if repo.MirrorNextSyncAt != nil {
		return true
	}

	return repo.MirrorInterval != "" && iv > 0
}

func (uc *RepositoryUseCase) syncAllPullMirrors(ctx context.Context) {
	global := uc.mirrorConfig().PullInterval
	now := time.Now()

	const page = 200
	offset := 0

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		schedules, err := uc.store.ListMirrorSchedules(ctx, domain.MirrorScheduleFilter{
			Offset:    offset,
			Limit:     page,
			DueBefore: &now,
		})
		if err != nil {
			slog.Error("list mirror schedules", "err", err)

			return
		}

		due := make([]uint, 0, len(schedules))
		for i := range schedules {
			if scheduleDue(&schedules[i], global, now) {
				due = append(due, schedules[i].ID)
			}
		}

		uc.syncBatch(ctx, due, domain.MirrorTriggerSchedule, 0)

		if len(schedules) < page {
			return
		}

		offset += page
	}
}

func scheduleDue(s *domain.MirrorSchedule, global time.Duration, now time.Time) bool {
	repo := domain.Repository{
		MirrorDirection:  s.Direction,
		MirrorURL:        s.URL,
		MirrorEnabled:    s.Enabled,
		MirrorInterval:   s.Interval,
		MirrorSyncWindow: s.SyncWindow,
		MirrorNextSyncAt: s.NextSyncAt,
	}

	return mirrorDue(&repo, global, now)
}

func (uc *RepositoryUseCase) updateLag(repo *domain.Repository, now time.Time) {
	if repo.MirrorDirection == domain.MirrorNone || repo.MirrorLastSyncAt == nil {
		return
	}

	uc.mirror.metrics.SetLagSeconds(repo.ID, now.Sub(*repo.MirrorLastSyncAt).Seconds())
}

func (uc *RepositoryUseCase) syncBatch(ctx context.Context, ids []uint, trigger string, actorID uint) {
	if len(ids) == 0 {
		return
	}

	workers := min(uc.mirrorConfig().Workers, len(ids))
	jobs := make(chan uint)

	var wg sync.WaitGroup
	for range workers {

		wg.Go(func() {

			for id := range jobs {
				if ctx.Err() != nil {
					continue
				}

				_, err := uc.syncMirrorByID(ctx, syncRequest{RepoID: id, Trigger: trigger, ActorID: actorID})
				if err != nil && !isBenignMirrorErr(err) {
					slog.Warn("mirror sync", "repo_id", id, "trigger", trigger, "err", err)
				}
			}
		})
	}

	for _, id := range ids {
		jobs <- id
	}

	close(jobs)
	wg.Wait()
}

func (uc *RepositoryUseCase) syncMirrorByID(ctx context.Context, req syncRequest) (*domain.Repository, error) {
	repo, err := uc.store.FindByID(ctx, req.RepoID)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}

	if repo.MirrorDirection == domain.MirrorNone || repo.MirrorURL == "" {
		return repo, ErrMirrorNotConfigured
	}

	if req.WantDir != "" && repo.MirrorDirection != req.WantDir {
		return repo, ErrMirrorNotConfigured
	}

	if !repo.MirrorEnabled {
		uc.markPaused(ctx, repo)

		return repo, ErrMirrorPaused
	}

	if req.Trigger == domain.MirrorTriggerSchedule && !inSyncWindow(repo.MirrorSyncWindow, time.Now()) {
		return repo, ErrMirrorSyncSkipped
	}

	lock := uc.mirrorLock(repo.ID)
	if !lock.TryLock() {
		uc.recordSkipped(ctx, repo, req, "sync already running")

		if req.Trigger == domain.MirrorTriggerPush || req.Trigger == domain.MirrorTriggerMerge {
			uc.mirror.mu.Lock()
			uc.mirror.pending[repo.ID] = true
			uc.mirror.mu.Unlock()
		}

		return repo, ErrMirrorSyncSkipped
	}

	slots, err := uc.acquireSyncSlot(ctx)
	if err != nil {
		lock.Unlock()

		return repo, err
	}

	defer uc.flushPendingPush(repo.ID)
	defer uc.releaseSyncSlot(slots)
	defer lock.Unlock()

	return repo, uc.runMirrorSync(ctx, repo, req)
}

func (uc *RepositoryUseCase) markPaused(ctx context.Context, repo *domain.Repository) {
	if repo.MirrorStatus == domain.MirrorStatusPaused {
		return
	}

	repo.MirrorStatus = domain.MirrorStatusPaused
	_ = uc.store.UpdateMirrorState(ctx, repo)
}

func (uc *RepositoryUseCase) recordSkipped(ctx context.Context, repo *domain.Repository, req syncRequest, reason string) {
	_ = ctx
	uc.mirror.metrics.ObserveSync(repo.MirrorDirection, metrics.ResultSkipped, 0)
	slog.Info("mirror sync skipped", "repo_id", repo.ID, "direction", repo.MirrorDirection, "trigger", req.Trigger, "reason", reason)
}

func (uc *RepositoryUseCase) createEvent(ctx context.Context, ev *domain.MirrorSyncEvent) {
	if uc.mirror.events == nil {
		return
	}

	if err := uc.mirror.events.Create(ctx, ev); err != nil {
		slog.Warn("mirror history create", "repo_id", ev.RepoID, "err", err)

		return
	}

	if err := uc.mirror.events.Prune(ctx, ev.RepoID, mirrorHistoryKeep); err != nil {
		slog.Warn("mirror history prune", "repo_id", ev.RepoID, "err", err)
	}
}

type progressCapture struct {
	mu   sync.Mutex
	last string
}

func (p *progressCapture) Write(b []byte) (int, error) {
	text := strings.NewReplacer("\r", "\n").Replace(string(b))

	for line := range strings.SplitSeq(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			p.mu.Lock()
			p.last = truncateErr(line, 200)
			p.mu.Unlock()
		}
	}

	return len(b), nil
}

func (p *progressCapture) Last() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.last
}

func mirrorAuth(username, password string) *domain.MirrorAuth {
	username = strings.TrimSpace(username)
	if username == "" && password == "" {
		return nil
	}

	return &domain.MirrorAuth{Username: username, Password: password}
}

func (uc *RepositoryUseCase) mirrorOptions(repo *domain.Repository) (*domain.MirrorOptions, error) {
	if repo.MirrorLFS {
		return nil, ErrMirrorLFSUnavailable
	}

	if repo.MirrorPartial {
		return nil, ErrMirrorPartialUnavailable
	}

	pw, err := uc.openSecret(repo.MirrorPassword)
	if err != nil {
		return nil, err
	}

	key, err := uc.openSecret(repo.MirrorSSHPrivateKey)
	if err != nil {
		return nil, err
	}

	pass, err := uc.openSecret(repo.MirrorSSHPassphrase)
	if err != nil {
		return nil, err
	}

	pins, err := uc.urlPolicy().parsePinnedHosts(repo.MirrorPinnedHosts)
	if err != nil {
		return nil, err
	}

	force, prune := repo.MirrorForce, repo.MirrorPrune

	return &domain.MirrorOptions{
		Auth:          mirrorAuth(repo.MirrorUsername, pw),
		SSHPrivateKey: key,
		SSHPassphrase: pass,
		InsecureTLS:   repo.MirrorInsecureTLS,
		Force:         &force,
		Prune:         &prune,
		RefSpecs:      parseRefspecs(repo.MirrorRefspecs),
		LFS:           false,
		MaxBytes:      repo.MirrorMaxBytes,
		Partial:       false,
		PinnedHosts:   pins,
	}, nil
}

type refSnapshot struct {
	branches map[string]string
	tags     map[string]string
}

func (uc *RepositoryUseCase) snapshotRefs(path string) refSnapshot {
	s := refSnapshot{branches: map[string]string{}, tags: map[string]string{}}

	if refs, err := uc.git.ListBranches(path); err == nil {
		for _, r := range refs {
			s.branches[r.Name] = r.CommitSHA
		}
	}

	if refs, err := uc.git.ListTags(path); err == nil {
		for _, r := range refs {
			s.tags[r.Name] = r.CommitSHA
		}
	}

	return s
}

func countChanged(before, after map[string]string) int {
	n := 0

	for name, sha := range after {
		if before[name] != sha {
			n++
		}
	}

	for name := range before {
		if _, ok := after[name]; !ok {
			n++
		}
	}

	return n
}

func (uc *RepositoryUseCase) afterPull(ctx context.Context, repo *domain.Repository, before refSnapshot, pc *progressCapture) error {
	_ = uc.git.EnsureDefaultBranch(repo.Path, repo.DefaultBranch)
	after := uc.snapshotRefs(repo.Path)

	if detected, err := uc.git.DetectHEADBranch(repo.Path); err == nil && detected != "" && validateBranchName(detected) == nil {
		if _, ok := after.branches[detected]; ok {
			repo.DefaultBranch = detected
		}
	}

	repo.MirrorRemoteHEAD = after.branches[repo.DefaultBranch]

	if repo.MirrorVerifySigned {
		if tip := repo.MirrorRemoteHEAD; tip != "" {
			if err := uc.git.VerifyCommitSignature(ctx, repo.Path, tip); err != nil {
				return fmt.Errorf("%w: %v", domain.ErrMirrorUnsigned, err)
			}
		}
	}

	changed := countChanged(before.branches, after.branches) + countChanged(before.tags, after.tags)
	progress := fmt.Sprintf("%d branches, %d tags, %d updated", len(after.branches), len(after.tags), changed)
	if last := pc.Last(); last != "" {
		progress += "; remote: " + last
	}

	repo.MirrorLastProgress = truncateErr(progress, 500)

	return nil
}

func (uc *RepositoryUseCase) performSync(ctx context.Context, repo *domain.Repository) error {
	if err := uc.validateMirrorURL(repo.MirrorURL); err != nil {
		return err
	}

	opts, err := uc.mirrorOptions(repo)
	if err != nil {
		return err
	}

	if isSSHMirrorURL(repo.MirrorURL) && strings.TrimSpace(opts.SSHPrivateKey) == "" {
		return ErrInvalidSSHKey
	}

	cfg := uc.mirrorConfig()
	pc := &progressCapture{}
	opts.Progress = pc

	switch repo.MirrorDirection {
	case domain.MirrorPull:
		sctx, cancel := context.WithTimeout(ctx, cfg.FetchTimeout)
		defer cancel()

		before := uc.snapshotRefs(repo.Path)
		if err := uc.git.MirrorFetch(sctx, repo.Path, repo.MirrorURL, opts); err != nil {
			return err
		}

		if err := uc.afterPull(sctx, repo, before, pc); err != nil {
			return err
		}
	case domain.MirrorPush:
		sctx, cancel := context.WithTimeout(ctx, cfg.PushTimeout)
		defer cancel()

		if repo.MirrorConflictPolicy == domain.MirrorConflictReject {
			f := false
			opts.Force = &f
		}

		if err := uc.git.MirrorPush(sctx, repo.Path, repo.MirrorURL, opts); err != nil {
			return err
		}

		snap := uc.snapshotRefs(repo.Path)
		repo.MirrorLastProgress = fmt.Sprintf("pushed %d branches, %d tags", len(snap.branches), len(snap.tags))
	default:
		return ErrInvalidMirrorDirection
	}

	return nil
}

func (uc *RepositoryUseCase) sanitizeMirrorErr(repo *domain.Repository, rawURL string, err error) string {
	msg := err.Error()

	for _, u := range []string{rawURL, repo.MirrorURL} {
		if u != "" {
			msg = strings.ReplaceAll(msg, u, domain.RedactURL(u))
		}
	}

	for _, stored := range []string{repo.MirrorPassword, repo.MirrorSSHPassphrase} {
		if plain, oerr := uc.openSecret(stored); oerr == nil && plain != "" {
			msg = strings.ReplaceAll(msg, plain, "***")
		}
	}

	return msg
}

func (uc *RepositoryUseCase) runMirrorSync(ctx context.Context, repo *domain.Repository, req syncRequest) error {
	started := time.Now().UTC()
	ev := &domain.MirrorSyncEvent{
		RepoID:    repo.ID,
		Direction: repo.MirrorDirection,
		Trigger:   req.Trigger,
		Status:    domain.MirrorEventRunning,
		StartedAt: started,
		ActorID:   req.ActorID,
	}

	uc.createEvent(ctx, ev)

	repo.MirrorStatus = domain.MirrorStatusSyncing

	syncErr := uc.performSync(ctx, repo)

	return uc.finishSync(ctx, repo, ev, started, syncErr, req)
}

func (uc *RepositoryUseCase) finishSync(ctx context.Context, repo *domain.Repository, ev *domain.MirrorSyncEvent, started time.Time, syncErr error, req syncRequest) error {
	pctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()

	now := time.Now().UTC()
	dur := now.Sub(started)
	global := uc.mirrorConfig().PullInterval
	iv := effectiveMirrorInterval(repo, global)

	var msg, class string
	if syncErr != nil {
		msg = truncateErr(uc.sanitizeMirrorErr(repo, repo.MirrorURL, syncErr), 500)
		class = classifyMirrorErr(syncErr)

		if errors.Is(syncErr, domain.ErrMirrorDiverged) && repo.MirrorDirection == domain.MirrorPush {
			msg = "push rejected: remote has diverged and conflict policy is reject"
		}

		repo.MirrorFailCount++
		repo.MirrorLastError = msg
		repo.MirrorStatus = domain.MirrorStatusError
		next := now.Add(mirrorBackoff(repo.MirrorFailCount))
		repo.MirrorNextSyncAt = &next
	} else {
		repo.MirrorFailCount = 0
		repo.MirrorLastError = ""
		repo.MirrorStatus = domain.MirrorStatusOK
		repo.MirrorLastSyncAt = &now
		repo.MirrorNextSyncAt = nil

		if iv > 0 {
			next := now.Add(iv)
			repo.MirrorNextSyncAt = &next
		}
	}

	if cur, err := uc.store.FindByID(pctx, repo.ID); err == nil && cur != nil && !cur.MirrorEnabled {
		repo.MirrorStatus = domain.MirrorStatusPaused
	}

	if err := uc.store.UpdateMirrorState(pctx, repo); err != nil {
		slog.Error("mirror save state", "repo_id", repo.ID, "err", err)
	}

	ev.FinishedAt = &now
	ev.DurationMs = dur.Milliseconds()
	ev.Status = domain.MirrorEventOK
	ev.Error = ""

	result := metrics.ResultOK
	if syncErr != nil {
		ev.Status = domain.MirrorEventFailed
		ev.Error = msg
		result = metrics.ResultFail
	}

	uc.saveEvent(pctx, ev)
	uc.mirror.metrics.ObserveSync(repo.MirrorDirection, result, dur)
	uc.updateLag(repo, now)
	uc.logSync(repo, ev, class, msg)

	if req.ActorID != 0 {
		summary := activitySummary("synced %s mirror (%s)", repo.MirrorDirection, req.Trigger)
		if syncErr != nil {
			summary = activitySummary("%s mirror sync failed (%s)", repo.MirrorDirection, req.Trigger)
		}

		recordRepoActivity(uc.activity, repo, req.ActorID, domain.ActivityMirrorSync, summary)
	}

	if syncErr != nil {
		return fmt.Errorf("%w: %s", ErrMirrorFailed, msg)
	}

	return nil
}

func (uc *RepositoryUseCase) saveEvent(ctx context.Context, ev *domain.MirrorSyncEvent) {
	if uc.mirror.events == nil || ev.ID == 0 {
		return
	}

	if err := uc.mirror.events.Update(ctx, ev); err != nil {
		slog.Warn("mirror history update", "repo_id", ev.RepoID, "err", err)
	}

	_ = uc.mirror.events.Prune(ctx, ev.RepoID, mirrorHistoryKeep)
}

func (uc *RepositoryUseCase) logSync(repo *domain.Repository, ev *domain.MirrorSyncEvent, class, msg string) {
	attrs := []any{
		"repo_id", repo.ID,
		"repo", repo.RepoPath,
		"direction", ev.Direction,
		"trigger", ev.Trigger,
		"status", ev.Status,
		"duration_ms", ev.DurationMs,
		"remote_host", mirrorHost(repo.MirrorURL),
	}

	if ev.Status == domain.MirrorEventFailed {
		attrs = append(attrs, "error_class", class, "err", msg, "fail_count", repo.MirrorFailCount)
		slog.Warn("mirror sync failed", attrs...)

		return
	}

	slog.Info("mirror sync ok", attrs...)
}

func mirrorHost(raw string) string {
	if _, host, ok := parseMirrorHost(raw); ok {
		return host
	}

	if u, err := url.Parse(raw); err == nil {
		return u.Hostname()
	}

	return ""
}

func classifyMirrorErr(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, domain.ErrMirrorDiverged):
		return "diverged"
	case errors.Is(err, domain.ErrMirrorTooLarge):
		return "too_large"
	case errors.Is(err, domain.ErrMirrorLFSMissing), errors.Is(err, domain.ErrMirrorLFSUnavailable), errors.Is(err, ErrMirrorLFSUnavailable):
		return "lfs"
	case errors.Is(err, domain.ErrMirrorPartialUnavailable), errors.Is(err, ErrMirrorPartialUnavailable):
		return "partial"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	}

	m := strings.ToLower(err.Error())
	switch {
	case strings.Contains(m, "authentication") || strings.Contains(m, "authorization") || strings.Contains(m, "unauthorized") || strings.Contains(m, "permission denied") || strings.Contains(m, "403") || strings.Contains(m, "401"):
		return "auth"
	case strings.Contains(m, "timeout") || strings.Contains(m, "no such host") || strings.Contains(m, "connection refused") || strings.Contains(m, "dial") || strings.Contains(m, "eof") || strings.Contains(m, "tls"):
		return "network"
	case strings.Contains(m, "not found") || strings.Contains(m, "404"):
		return "not_found"
	case strings.Contains(m, "lfs"):
		return "lfs"
	default:
		return "other"
	}
}

func (uc *RepositoryUseCase) mirrorLock(repoID uint) *sync.Mutex {
	uc.mirror.mu.Lock()
	defer uc.mirror.mu.Unlock()

	if uc.mirror.locks[repoID] == nil {
		uc.mirror.locks[repoID] = &sync.Mutex{}
	}

	return uc.mirror.locks[repoID]
}
