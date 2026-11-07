package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

const mirrorCheckRefsCap = 50

type MirrorStatusItem struct {
	Direction     string     `json:"direction"`
	URL           string     `json:"url"`
	Enabled       bool       `json:"enabled"`
	Status        string     `json:"status"`
	ReadOnly      bool       `json:"read_only"`
	LastSyncAt    *time.Time `json:"last_sync_at,omitzero"`
	LagSeconds    *int64     `json:"lag_seconds,omitzero"`
	Progress      string     `json:"progress,omitzero"`
	RemoteHead    string     `json:"remote_head,omitzero"`
	DefaultBranch string     `json:"default_branch"`
	Interval      string     `json:"interval,omitzero"`
	SyncWindow    string     `json:"sync_window,omitzero"`

	LastError      string     `json:"last_error,omitzero"`
	NextSyncAt     *time.Time `json:"next_sync_at,omitzero"`
	FailCount      int        `json:"fail_count,omitzero"`
	Force          *bool      `json:"force,omitzero"`
	Prune          *bool      `json:"prune,omitzero"`
	Refspecs       string     `json:"refspecs,omitzero"`
	LFS            *bool      `json:"lfs,omitzero"`
	MaxBytes       int64      `json:"max_bytes,omitzero"`
	ConflictPolicy string     `json:"conflict_policy,omitzero"`
	HasSSHKey      *bool      `json:"has_ssh_key,omitzero"`
	InsecureTLS    *bool      `json:"insecure_tls,omitzero"`
	InsecureNotice string     `json:"insecure_tls_warning,omitzero"`
	PinnedHosts    string     `json:"pinned_hosts,omitzero"`
	OwnerOnly      *bool      `json:"owner_only,omitzero"`
	Warnings       []string   `json:"warnings,omitzero"`
}

type MirrorHistoryItem struct {
	ID         uint       `json:"id"`
	Direction  string     `json:"direction"`
	Trigger    string     `json:"trigger"`
	Status     string     `json:"status"`
	Error      string     `json:"error,omitzero"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitzero"`
	DurationMs int64      `json:"duration_ms"`
	ActorID    uint       `json:"actor_id,omitzero"`
}

type CheckMirrorInput struct {
	ResolveRepositoryInput
	URL      string
	Username *string
	Password *string
	DryRun   bool
}

type MirrorCheckResult struct {
	OK            bool             `json:"ok"`
	Error         string           `json:"error,omitzero"`
	DefaultBranch string           `json:"default_branch,omitzero"`
	Head          string           `json:"head,omitzero"`
	RefCount      int              `json:"ref_count"`
	Refs          []domain.RefInfo `json:"refs,omitzero"`
	WouldUpdate   []domain.RefInfo `json:"would_update,omitzero"`
}

func effectiveMirrorStatus(repo *domain.Repository) string {
	if !repo.MirrorEnabled {
		return domain.MirrorStatusPaused
	}

	if repo.MirrorStatus == "" {
		return domain.MirrorStatusIdle
	}

	return repo.MirrorStatus
}

func (uc *RepositoryUseCase) mirrorStatusFromRepo(repo *domain.Repository, canAdmin bool, now time.Time) *MirrorStatusItem {
	st := &MirrorStatusItem{
		Direction: repo.MirrorDirection,
		Enabled:   repo.MirrorEnabled,
		Status:    effectiveMirrorStatus(repo),
		ReadOnly:  repo.MirrorDirection == domain.MirrorPull,
	}

	if !canAdmin {
		return st
	}

	st.URL = domain.RedactURL(repo.MirrorURL)
	st.LastSyncAt = repo.MirrorLastSyncAt
	st.Progress = repo.MirrorLastProgress
	st.RemoteHead = repo.MirrorRemoteHEAD
	st.DefaultBranch = repo.DefaultBranch
	st.Interval = repo.MirrorInterval
	st.SyncWindow = repo.MirrorSyncWindow

	if repo.MirrorLastSyncAt != nil {
		lag := int64(now.Sub(*repo.MirrorLastSyncAt).Seconds())
		st.LagSeconds = &lag
	}

	uc.fillMirrorAdminStatus(st, repo)
	return st
}

func (uc *RepositoryUseCase) fillMirrorAdminStatus(st *MirrorStatusItem, repo *domain.Repository) {
	force, prune, lfs := repo.MirrorForce, repo.MirrorPrune, repo.MirrorLFS
	hasKey, insecure, ownerOnly := repo.MirrorSSHPrivateKey != "", repo.MirrorInsecureTLS, repo.MirrorOwnerOnly

	st.LastError = repo.MirrorLastError
	st.NextSyncAt = repo.MirrorNextSyncAt
	st.FailCount = repo.MirrorFailCount
	st.Force, st.Prune, st.LFS = &force, &prune, &lfs
	st.Refspecs = repo.MirrorRefspecs
	st.MaxBytes = repo.MirrorMaxBytes
	st.ConflictPolicy = repo.MirrorConflictPolicy
	st.HasSSHKey, st.InsecureTLS, st.OwnerOnly = &hasKey, &insecure, &ownerOnly
	st.PinnedHosts = repo.MirrorPinnedHosts

	if insecure {
		st.InsecureNotice = "TLS certificate verification is disabled for this mirror (HTTPS only; SSH host keys still verified)"
		st.Warnings = append(st.Warnings, st.InsecureNotice)
	}

	if prune && repo.MirrorDirection == domain.MirrorPull {
		st.Warnings = append(st.Warnings, "prune is enabled: refs deleted upstream are deleted locally")
	}
}

func (uc *RepositoryUseCase) GetMirrorStatus(ctx context.Context, in ResolveRepositoryInput) (*MirrorStatusItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, _, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, err
	}

	if repo.MirrorDirection == domain.MirrorNone {
		return nil, ErrMirrorNotConfigured
	}

	return uc.mirrorStatusFromRepo(repo, uc.authz.CanAdminRepo(repo, in.ViewerID), time.Now()), nil
}

func (uc *RepositoryUseCase) MirrorHistory(ctx context.Context, in ResolveRepositoryInput, limit int) ([]MirrorHistoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, _, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	if uc.mirror.events == nil {
		return []MirrorHistoryItem{}, nil
	}

	events, err := uc.mirror.events.ListByRepo(ctx, repo.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("mirror history: %w", err)
	}

	out := make([]MirrorHistoryItem, 0, len(events))
	for _, e := range events {
		out = append(out, MirrorHistoryItem{
			ID:         e.ID,
			Direction:  e.Direction,
			Trigger:    e.Trigger,
			Status:     e.Status,
			Error:      e.Error,
			StartedAt:  e.StartedAt,
			FinishedAt: e.FinishedAt,
			DurationMs: e.DurationMs,
			ActorID:    e.ActorID,
		})
	}

	return out, nil
}

func (uc *RepositoryUseCase) CheckMirror(ctx context.Context, in CheckMirrorInput) (*MirrorCheckResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, _, err := uc.resolve(in.ResolveRepositoryInput)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	target := strings.TrimSpace(in.URL)
	if target == "" {
		target = repo.MirrorURL
	}

	if target == "" {
		return nil, ErrMirrorNotConfigured
	}

	if err := uc.validateMirrorURL(target); err != nil {
		return nil, err
	}

	if !uc.mirrorLimiter().allow(repo.ID|checkLimiterBit, time.Now()) {
		return nil, ErrMirrorRateLimited
	}

	opts, err := uc.mirrorOptions(repo)
	if err != nil {
		return nil, err
	}

	if in.Username != nil || in.Password != nil {
		opts.Auth = overrideAuth(opts.Auth, in.Username, in.Password)
	}

	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return uc.runMirrorCheck(cctx, repo, target, opts, in.DryRun), nil
}

const checkLimiterBit = uint(1) << 40

func overrideAuth(cur *domain.MirrorAuth, user, pass *string) *domain.MirrorAuth {
	out := domain.MirrorAuth{}
	if cur != nil {
		out = *cur
	}

	if user != nil {
		out.Username = strings.TrimSpace(*user)
	}

	if pass != nil {
		out.Password = *pass
	}

	return &out
}

func (uc *RepositoryUseCase) runMirrorCheck(ctx context.Context, repo *domain.Repository, target string, opts *domain.MirrorOptions, dryRun bool) *MirrorCheckResult {
	if dryRun && repo.MirrorDirection == domain.MirrorPull {
		info, updates, err := uc.git.MirrorDryRunFetch(ctx, repo.Path, target, opts)
		if err != nil {
			return &MirrorCheckResult{Error: truncateErr(uc.sanitizeMirrorErr(repo, target, err), 500)}
		}

		res := &MirrorCheckResult{OK: true}
		if info != nil {
			res.DefaultBranch = info.DefaultBranch
			res.Head = info.Head
			res.RefCount = len(info.Refs)
			res.Refs = info.Refs
		}
		if len(res.Refs) > mirrorCheckRefsCap {
			res.Refs = res.Refs[:mirrorCheckRefsCap]
		}

		if len(updates) > mirrorCheckRefsCap {
			updates = updates[:mirrorCheckRefsCap]
		}

		res.WouldUpdate = updates

		return res
	}

	info, err := uc.git.MirrorCheck(ctx, target, opts)
	if err != nil {
		return &MirrorCheckResult{
			Error: truncateErr(uc.sanitizeMirrorErr(repo, target, err), 500),
		}
	}

	res := &MirrorCheckResult{
		OK:            true,
		DefaultBranch: info.DefaultBranch,
		Head:          info.Head,
		RefCount:      len(info.Refs),
		Refs:          info.Refs,
	}
	if len(res.Refs) > mirrorCheckRefsCap {
		res.Refs = res.Refs[:mirrorCheckRefsCap]
	}

	return res
}
