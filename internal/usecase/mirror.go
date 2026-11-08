package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrInvalidMirrorDirection    = NewError(KindValidation, "invalid_mirror_direction")
	ErrMirrorNotConfigured       = NewError(KindValidation, "mirror_not_configured")
	ErrMirrorReadOnly            = NewError(KindForbidden, "mirror_read_only")
	ErrMirrorOwnerOnly           = NewError(KindForbidden, "mirror_owner_only")
	ErrMirrorAdminOnly           = NewError(KindForbidden, "mirror_admin_only")
	ErrMirrorHTTPForbidden       = NewError(KindValidation, "mirror_http_forbidden")
	ErrMirrorVerifyNoTrust       = NewError(KindValidation, "mirror_verify_no_trust")
	ErrMirrorPinnedHostsRequired = NewError(KindValidation, "mirror_pinned_hosts_required")
)

type ConfigureMirrorInput struct {
	ViewerID        uint
	Name            string
	Direction       string
	URL             string
	Username        *string
	Password        *string
	ClearPassword   bool
	Enabled         bool
	Clear           bool
	Interval        *string
	SyncWindow      *string
	Force           *bool
	Prune           *bool
	Refspecs        *string
	SSHPrivateKey   *string
	SSHPassphrase   *string
	ClearSSHKey     bool
	InsecureTLS     *bool
	PinnedHosts     *string
	MaxBytes        *int64
	ConflictPolicy  *string
	OwnerOnly       *bool
	GroupID         *uint
	VerifySigned    *bool
	CurrentPassword string
}

type mirrorTrustChecker interface {
	VerifyTrustConfigured() bool
}

func (uc *RepositoryUseCase) requireVerifyTrust() error {
	if checker, ok := uc.git.(mirrorTrustChecker); ok && !checker.VerifyTrustConfigured() {
		return ErrMirrorVerifyNoTrust
	}

	return nil
}

func mirrorNeedsStepUp(repo *domain.Repository, in ConfigureMirrorInput) bool {
	if u := strings.TrimSpace(in.URL); u != "" && u != repo.MirrorURL {
		return true
	}

	if in.ClearPassword || in.ClearSSHKey {
		return true
	}

	if in.Password != nil && *in.Password != "" {
		return true
	}

	if in.SSHPrivateKey != nil && strings.TrimSpace(*in.SSHPrivateKey) != "" {
		return true
	}

	if in.SSHPassphrase != nil {
		return true
	}

	return mirrorBehaviorChanges(repo, in)
}

func mirrorBehaviorChanges(repo *domain.Repository, in ConfigureMirrorInput) bool {
	if dir := strings.ToLower(strings.TrimSpace(in.Direction)); dir != "" && dir != repo.MirrorDirection {
		return true
	}

	if in.Refspecs != nil {
		specs, err := normalizeRefspecs(*in.Refspecs)
		if err != nil || specs != repo.MirrorRefspecs {
			return true
		}
	}

	if in.PinnedHosts != nil && strings.Join(splitList(*in.PinnedHosts), ",") != repo.MirrorPinnedHosts {
		return true
	}

	if in.ConflictPolicy != nil {
		policy, err := normalizeConflictPolicy(*in.ConflictPolicy)
		current := repo.MirrorConflictPolicy
		if current == "" {
			current = domain.MirrorConflictForce
		}

		if err != nil || policy != current {
			return true
		}
	}

	if in.Force != nil && *in.Force != repo.MirrorForce {
		return true
	}

	if in.Prune != nil && *in.Prune != repo.MirrorPrune {
		return true
	}

	return in.InsecureTLS != nil && *in.InsecureTLS != repo.MirrorInsecureTLS
}

func (uc *RepositoryUseCase) viewerIsSiteAdmin(ctx context.Context, viewerID uint) bool {
	if viewerID == 0 || uc.users == nil {
		return false
	}

	u, err := uc.users.FindByID(ctx, viewerID)

	return err == nil && u != nil && u.IsAdmin()
}

func (uc *RepositoryUseCase) requireSiteAdmin(ctx context.Context, viewerID uint) error {
	if viewerID == 0 {
		return ErrUnauthorized
	}

	if !uc.viewerIsSiteAdmin(ctx, viewerID) {
		return ErrAdminRequired
	}

	return nil
}

func (uc *RepositoryUseCase) requireMirrorMutate(ctx context.Context, repo *domain.Repository, viewerID uint) error {
	if !uc.authz.CanAdminRepo(repo, viewerID) {
		return ErrRepoNotFound
	}

	if repo.MirrorOwnerOnly && !isPersonalOwner(repo, viewerID) && !uc.viewerIsSiteAdmin(ctx, viewerID) {
		return ErrMirrorOwnerOnly
	}

	return nil
}

func (uc *RepositoryUseCase) ConfigureMirror(ctx context.Context, in ConfigureMirrorInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, err := uc.resolve(ctx, ResolveRepositoryInput{
		Name:     in.Name,
		ViewerID: in.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.requireMirrorMutate(ctx, repo, in.ViewerID); err != nil {
		return nil, err
	}

	if !in.Clear && mirrorNeedsStepUp(repo, in) {
		if err := uc.requireViewerPassword(ctx, in.ViewerID, in.CurrentPassword); err != nil {
			return nil, err
		}
	}

	siteAdmin := uc.viewerIsSiteAdmin(ctx, in.ViewerID)
	isOwner := isPersonalOwner(repo, in.ViewerID)

	lock := uc.mirrorLock(repo.ID)
	if !lock.TryLock() {
		return nil, ErrMirrorBusy
	}
	defer lock.Unlock()

	if in.Clear {
		return uc.clearMirror(ctx, repo, ownerSlug, in.ViewerID)
	}

	if !in.Enabled && repo.MirrorDirection == domain.MirrorNone && strings.TrimSpace(in.URL) == "" {
		item := uc.toRepositoryItem(repo, ownerSlug, in.ViewerID)

		return &item, nil
	}

	prev := *repo
	if err := uc.applyMirrorCore(repo, in); err != nil {
		return nil, err
	}

	if err := uc.applyMirrorTuning(repo, in, siteAdmin, isOwner || siteAdmin); err != nil {
		return nil, err
	}

	if err := uc.requireSSHKeyIfNeeded(repo); err != nil {
		return nil, err
	}

	repo.MirrorEnabled = in.Enabled
	uc.settleMirrorStatus(repo, &prev)

	if err := uc.store.Update(repo); err != nil {
		return nil, fmt.Errorf("save mirror: %w", err)
	}

	uc.auditConfigure(repo, &prev, in.ViewerID)

	material := mirrorFingerprint(repo) != mirrorFingerprint(&prev) || (!prev.MirrorEnabled && repo.MirrorEnabled)
	if material && repo.MirrorEnabled && repo.MirrorDirection == domain.MirrorPull && uc.mirrorConfig().SyncOnConfigure {
		id, actor := repo.ID, in.ViewerID
		uc.goMirror(func() {
			ctx, cancel := context.WithTimeout(uc.mirrorRootContext(), uc.mirrorConfig().CloneTimeout)
			defer cancel()

			_, _ = uc.syncMirrorByID(ctx, syncRequest{RepoID: id, WantDir: domain.MirrorPull, Trigger: domain.MirrorTriggerConfig, ActorID: actor})
		})
	}

	item := uc.toRepositoryItem(repo, ownerSlug, in.ViewerID)

	return &item, nil
}

func (uc *RepositoryUseCase) clearMirror(ctx context.Context, repo *domain.Repository, ownerSlug string, viewerID uint) (*RepositoryItem, error) {
	prev := *repo
	resetMirrorConfig(repo)

	if err := uc.store.Update(repo); err != nil {
		return nil, fmt.Errorf("clear mirror: %w", err)
	}

	uc.releaseMirrorRuntime(repo.ID)
	uc.auditConfigure(repo, &prev, viewerID)

	item := uc.toRepositoryItem(repo, ownerSlug, viewerID)

	return &item, nil
}

func (uc *RepositoryUseCase) releaseMirrorRuntime(repoID uint) {
	uc.mirror.mu.Lock()
	if t, ok := uc.mirror.debounce[repoID]; ok {
		t.Stop()
		delete(uc.mirror.debounce, repoID)
	}
	delete(uc.mirror.locks, repoID)
	delete(uc.mirror.pending, repoID)
	metrics := uc.mirror.metrics
	uc.mirror.mu.Unlock()

	if metrics != nil {
		metrics.ForgetRepo(repoID)
	}
}

func resetMirrorConfig(repo *domain.Repository) {
	repo.MirrorDirection = domain.MirrorNone
	repo.MirrorURL = ""
	repo.MirrorUsername = ""
	repo.MirrorPassword = ""
	repo.MirrorSSHPrivateKey = ""
	repo.MirrorSSHPassphrase = ""
	repo.MirrorEnabled = true
	repo.MirrorInterval = ""
	repo.MirrorSyncWindow = ""
	repo.MirrorForce = true
	repo.MirrorPrune = false
	repo.MirrorRefspecs = ""
	repo.MirrorInsecureTLS = false
	repo.MirrorPinnedHosts = ""
	repo.MirrorVerifySigned = false
	repo.MirrorMaxBytes = 0
	repo.MirrorConflictPolicy = ""
	repo.MirrorOwnerOnly = false
	repo.MirrorGroupID = 0
	resetMirrorState(repo)
}

func resetMirrorState(repo *domain.Repository) {
	repo.MirrorLastSyncAt = nil
	repo.MirrorLastError = ""
	repo.MirrorStatus = domain.MirrorStatusIdle
	repo.MirrorFailCount = 0
	repo.MirrorNextSyncAt = nil
	repo.MirrorLastProgress = ""
	repo.MirrorRemoteHEAD = ""

	if repo.MirrorDirection == domain.MirrorNone {
		repo.MirrorStatus = ""
	}
}

func mirrorFingerprint(r *domain.Repository) string {
	return fmt.Sprint(r.MirrorDirection, "|", r.MirrorURL, "|", r.MirrorUsername, "|", r.MirrorPassword, "|", r.MirrorSSHPrivateKey, "|",
		r.MirrorSSHPassphrase, "|", r.MirrorRefspecs, "|", r.MirrorForce, "|", r.MirrorPrune, "|",
		r.MirrorInsecureTLS, "|", r.MirrorPinnedHosts)
}

func (uc *RepositoryUseCase) settleMirrorStatus(repo, prev *domain.Repository) {
	switch {
	case !repo.MirrorEnabled:
		repo.MirrorStatus = domain.MirrorStatusPaused
	case prev.MirrorStatus == domain.MirrorStatusPaused || repo.MirrorStatus == "" || repo.MirrorStatus == domain.MirrorStatusPaused:
		repo.MirrorStatus = domain.MirrorStatusIdle
		repo.MirrorNextSyncAt = nil
	}
}

func (uc *RepositoryUseCase) applyMirrorCore(repo *domain.Repository, in ConfigureMirrorInput) error {
	direction := strings.ToLower(strings.TrimSpace(in.Direction))
	if direction == "" {
		direction = repo.MirrorDirection
	}

	if direction != domain.MirrorPull && direction != domain.MirrorPush {
		return ErrInvalidMirrorDirection
	}

	mirrorURL := strings.TrimSpace(in.URL)
	if mirrorURL == "" {
		mirrorURL = repo.MirrorURL
	}

	if err := uc.validateMirrorURL(mirrorURL); err != nil {
		return err
	}

	fresh := repo.MirrorDirection == domain.MirrorNone
	changedDir := repo.MirrorDirection != direction
	changedURL := repo.MirrorURL != mirrorURL

	if fresh {
		repo.MirrorForce, repo.MirrorPrune = true, false
	}

	if repo.MirrorConflictPolicy == "" {
		repo.MirrorConflictPolicy = domain.MirrorConflictForce
	}

	if changedDir && direction == domain.MirrorPull && in.Prune == nil {
		if refs, err := uc.git.ListBranches(repo.Path); err == nil && len(refs) > 0 {
			repo.MirrorPrune = false
		}
	}

	repo.MirrorDirection = direction
	repo.MirrorURL = mirrorURL

	if changedDir || changedURL {
		resetMirrorState(repo)
		clearMirrorSecretsUnlessProvided(repo, in)
	}

	return uc.applyMirrorCredentials(repo, in)
}

func clearMirrorSecretsUnlessProvided(repo *domain.Repository, in ConfigureMirrorInput) {
	keepPassword := in.Password != nil && strings.TrimSpace(*in.Password) != "" && !in.ClearPassword
	keepSSH := in.SSHPrivateKey != nil && strings.TrimSpace(*in.SSHPrivateKey) != "" && !in.ClearSSHKey
	if !keepPassword {
		repo.MirrorPassword = ""
		if in.Username == nil {
			repo.MirrorUsername = ""
		}
	}

	if !keepSSH {
		repo.MirrorSSHPrivateKey = ""
		repo.MirrorSSHPassphrase = ""
	}
}

func (uc *RepositoryUseCase) applyMirrorCredentials(repo *domain.Repository, in ConfigureMirrorInput) error {
	if in.Username != nil {
		repo.MirrorUsername = strings.TrimSpace(*in.Username)
	}

	switch {
	case in.ClearPassword:
		repo.MirrorPassword = ""
	case in.Password != nil && *in.Password != "":
		sealed, err := uc.sealSecret(*in.Password)
		if err != nil {
			return err
		}

		repo.MirrorPassword = sealed
	}

	if in.ClearSSHKey {
		repo.MirrorSSHPrivateKey = ""
		repo.MirrorSSHPassphrase = ""
	}

	return uc.applyMirrorSSHKey(repo, in)
}

func (uc *RepositoryUseCase) applyMirrorSSHKey(repo *domain.Repository, in ConfigureMirrorInput) error {
	if in.SSHPrivateKey != nil && strings.TrimSpace(*in.SSHPrivateKey) != "" {
		pass := ""
		if in.SSHPassphrase != nil {
			pass = *in.SSHPassphrase
		}

		if err := validateSSHKey(*in.SSHPrivateKey, pass); err != nil {
			return err
		}

		key, err := uc.sealSecret(*in.SSHPrivateKey)
		if err != nil {
			return err
		}

		sealedPass, err := uc.sealSecret(pass)
		if err != nil {
			return err
		}

		repo.MirrorSSHPrivateKey, repo.MirrorSSHPassphrase = key, sealedPass

		return nil
	}

	if in.SSHPassphrase != nil && repo.MirrorSSHPrivateKey != "" {
		sealedPass, err := uc.sealSecret(*in.SSHPassphrase)
		if err != nil {
			return err
		}

		repo.MirrorSSHPassphrase = sealedPass
	}

	return nil
}

func (uc *RepositoryUseCase) applyMirrorTuning(repo *domain.Repository, in ConfigureMirrorInput, siteAdmin, ownerOrAdmin bool) error {
	if err := applyMirrorSchedule(repo, in); err != nil {
		return err
	}

	if in.Force != nil {
		repo.MirrorForce = *in.Force
	}

	if in.Prune != nil {
		repo.MirrorPrune = *in.Prune
	}

	if in.VerifySigned != nil {
		if *in.VerifySigned {
			if err := uc.requireVerifyTrust(); err != nil {
				return err
			}
		}

		repo.MirrorVerifySigned = *in.VerifySigned
	}

	if in.MaxBytes != nil {
		if *in.MaxBytes < 0 {
			return ErrInvalidMirrorOption
		}

		repo.MirrorMaxBytes = *in.MaxBytes
	}

	if in.Refspecs != nil {
		specs, err := normalizeRefspecs(*in.Refspecs)
		if err != nil {
			return err
		}

		repo.MirrorRefspecs = specs
	}

	if in.ConflictPolicy != nil {
		policy, err := normalizeConflictPolicy(*in.ConflictPolicy)
		if err != nil {
			return err
		}

		repo.MirrorConflictPolicy = policy
	}

	return uc.applyMirrorRestricted(repo, in, siteAdmin, ownerOrAdmin)
}

func applyMirrorSchedule(repo *domain.Repository, in ConfigureMirrorInput) error {
	if in.Interval != nil {
		iv, err := normalizeInterval(*in.Interval)
		if err != nil {
			return err
		}

		repo.MirrorInterval = iv
		repo.MirrorNextSyncAt = nil
	}

	if in.SyncWindow != nil {
		w, err := normalizeSyncWindow(*in.SyncWindow)
		if err != nil {
			return err
		}

		repo.MirrorSyncWindow = w
	}

	return nil
}

func (uc *RepositoryUseCase) applyMirrorRestricted(repo *domain.Repository, in ConfigureMirrorInput, siteAdmin, ownerOrAdmin bool) error {
	if in.PinnedHosts != nil {
		if _, err := uc.urlPolicy().parsePinnedHosts(*in.PinnedHosts); err != nil {
			return err
		}

		repo.MirrorPinnedHosts = strings.Join(splitList(*in.PinnedHosts), ",")
	}

	if in.InsecureTLS != nil {
		if *in.InsecureTLS && !siteAdmin {
			return ErrMirrorAdminOnly
		}

		repo.MirrorInsecureTLS = *in.InsecureTLS
	}

	if repo.MirrorInsecureTLS && strings.TrimSpace(repo.MirrorPinnedHosts) == "" {
		return ErrMirrorPinnedHostsRequired
	}

	if in.OwnerOnly != nil {
		if !ownerOrAdmin {
			return ErrMirrorOwnerOnly
		}

		repo.MirrorOwnerOnly = *in.OwnerOnly
	}

	if in.GroupID != nil {
		if gid := *in.GroupID; gid != 0 {
			if uc.mirror.groups == nil {
				return ErrMirrorGroupNotFound
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := uc.mirror.groups.FindByID(ctx, gid)
			cancel()
			if err != nil {
				return ErrMirrorGroupNotFound
			}
		}

		repo.MirrorGroupID = *in.GroupID
	}

	return nil
}

func (uc *RepositoryUseCase) auditConfigure(repo, prev *domain.Repository, actorID uint) {
	var what string

	switch {
	case repo.MirrorDirection == domain.MirrorNone && prev.MirrorDirection != domain.MirrorNone:
		what = "removed mirror"
	case prev.MirrorDirection == domain.MirrorNone:
		what = "enabled " + repo.MirrorDirection + " mirror"
	case repo.MirrorURL != prev.MirrorURL:
		what = "changed mirror URL"
	case prev.MirrorEnabled && !repo.MirrorEnabled:
		what = "paused mirror"
	case !prev.MirrorEnabled && repo.MirrorEnabled:
		what = "resumed mirror"
	default:
		what = "updated mirror settings"
	}

	slog.Info("mirror configured", "repo_id", repo.ID, "actor_id", actorID)
	if repo.MirrorInsecureTLS && !prev.MirrorInsecureTLS {
		slog.Warn("mirror insecure tls", "repo_id", repo.ID, "actor_id", actorID)
	} else if !repo.MirrorInsecureTLS && prev.MirrorInsecureTLS {
		slog.Info("mirror tls ok", "repo_id", repo.ID, "actor_id", actorID)
	}

	recordRepoActivity(uc.activity, repo, actorID, domain.ActivityMirrorConfigure, activitySummary("%s", what))
}

func (uc *RepositoryUseCase) SyncMirror(ctx context.Context, in ResolveRepositoryInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, err := uc.resolve(ctx, in)
	if err != nil {
		return nil, err
	}

	if err := uc.requireMirrorMutate(ctx, repo, in.ViewerID); err != nil {
		return nil, err
	}

	if repo.MirrorDirection == domain.MirrorNone || strings.TrimSpace(repo.MirrorURL) == "" {
		return nil, ErrMirrorNotConfigured
	}

	if !uc.mirrorLimiter().allow(repo.ID, time.Now()) {
		return nil, ErrMirrorRateLimited
	}

	fresh, err := uc.syncMirrorByID(ctx, syncRequest{RepoID: repo.ID, Trigger: domain.MirrorTriggerManual, ActorID: in.ViewerID})
	if err != nil {
		if errors.Is(err, ErrMirrorFailed) {
			return nil, ErrMirrorFailed
		}

		return nil, err
	}

	item := uc.toRepositoryItem(fresh, ownerSlug, in.ViewerID)

	return &item, nil
}

func (uc *RepositoryUseCase) mirrorLimiter() *rateLimiter {
	uc.mirror.mu.Lock()
	defer uc.mirror.mu.Unlock()

	return uc.mirror.limiter
}

func truncateErr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}

	return s[:n]
}

func normalizeMirrorDirection(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(raw))
	switch d {
	case "":
		return domain.MirrorNone, nil
	case domain.MirrorPull:
		return domain.MirrorPull, nil
	case domain.MirrorPush:
		return domain.MirrorPush, nil
	default:
		return "", ErrInvalidMirrorDirection
	}
}
