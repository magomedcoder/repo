package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

func pull(e *mirrorEnv, id uint) error {
	_, err := e.uc.syncMirrorByID(context.Background(), syncRequest{RepoID: id, Trigger: domain.MirrorTriggerManual})

	return err
}

func TestSyncSkippedWhenBusy(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)

	lock := e.uc.mirrorLock(repo.ID)
	lock.Lock()

	err := pull(e, repo.ID)
	lock.Unlock()

	if !errors.Is(err, ErrMirrorSyncSkipped) {
		t.Fatalf("want ErrMirrorSyncSkipped, got %v", err)
	}

	if evs := e.events.snapshot(); len(evs) != 0 {
		t.Fatalf("events=%+v", evs)
	}

	if e.git.fetches != 0 {
		t.Fatal("fetch must not run while busy")
	}
}

func TestSyncSuccessUsesStoredCredsAndRecords(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)

	var gotPW, gotUser string

	e.git.fetchFn = func(_ context.Context, _, _ string, o *domain.MirrorOptions) error {
		gotPW, gotUser = o.Auth.Password, o.Auth.Username

		return nil
	}

	if err := pull(e, repo.ID); err != nil {
		t.Fatal(err)
	}

	e.uc.WaitMirrorIdle()

	if gotPW != "s3cret-pw" || gotUser != "bot" {
		t.Fatalf("adapter must receive stored creds, got %q/%q", gotUser, gotPW)
	}

	cur := e.store.get(repo.ID)
	if cur.MirrorStatus != domain.MirrorStatusOK || cur.MirrorLastSyncAt == nil || cur.MirrorFailCount != 0 {
		t.Fatalf("state=%+v", cur)
	}

	if cur.MirrorRemoteHEAD != "aaa" || cur.MirrorLastProgress == "" {
		t.Fatalf("remote head/progress not recorded: %q %q", cur.MirrorRemoteHEAD, cur.MirrorLastProgress)
	}

	evs := e.events.snapshot()
	if len(evs) != 1 || evs[0].Status != domain.MirrorEventOK || evs[0].FinishedAt == nil {
		t.Fatalf("events=%+v", evs)
	}
}

func TestSyncFailureBackoffAndRedaction(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)

	e.git.fetchFn = func(_ context.Context, _, url string, _ *domain.MirrorOptions) error {
		return fmt.Errorf("auth failed for %s with password s3cret-pw", url)
	}

	want := []time.Duration{5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour, 6 * time.Hour}

	for i, d := range want {
		err := pull(e, repo.ID)
		if !errors.Is(err, ErrMirrorFailed) {
			t.Fatalf("round %d: %v", i, err)
		}

		cur := e.store.get(repo.ID)
		if cur.MirrorFailCount != i+1 || cur.MirrorStatus != domain.MirrorStatusError || cur.MirrorNextSyncAt == nil {
			t.Fatalf("round %d state=%+v", i, cur)
		}

		if got := time.Until(*cur.MirrorNextSyncAt); got < d-time.Minute || got > d+time.Minute {
			t.Fatalf("round %d: next in %s, want ~%s", i, got, d)
		}

		if strings.Contains(cur.MirrorLastError, "s3cret-pw") || strings.Contains(err.Error(), "s3cret-pw") {
			t.Fatalf("secret leaked: %q", cur.MirrorLastError)
		}
	}

	e.git.fetchFn = nil

	if err := pull(e, repo.ID); err != nil {
		t.Fatal(err)
	}

	if cur := e.store.get(repo.ID); cur.MirrorFailCount != 0 || cur.MirrorLastError != "" {
		t.Fatalf("success must reset failure state: %+v", cur)
	}
}

func TestSyncPausedDoesNothing(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)
	repo.MirrorEnabled = false
	_ = e.store.Update(repo)

	if err := pull(e, repo.ID); !errors.Is(err, ErrMirrorPaused) {
		t.Fatalf("want paused, got %v", err)
	}

	if e.git.fetches != 0 {
		t.Fatal("paused mirror must not fetch")
	}

	if cur := e.store.get(repo.ID); cur.MirrorStatus != domain.MirrorStatusPaused {
		t.Fatalf("status=%s", cur.MirrorStatus)
	}
}

func TestPushRejectPolicyDoesNotForce(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPush)
	repo.MirrorConflictPolicy = domain.MirrorConflictReject
	_ = e.store.Update(repo)

	var force *bool

	e.git.pushFn = func(_ context.Context, _, _ string, o *domain.MirrorOptions) error {
		force = o.Force

		return fmt.Errorf("%w: non-fast-forward", domain.ErrMirrorDiverged)
	}

	err := pull(e, repo.ID)
	if !errors.Is(err, ErrMirrorFailed) || !strings.Contains(err.Error(), "diverged") {
		t.Fatalf("err=%v", err)
	}

	if force == nil || *force {
		t.Fatal("reject policy must disable force")
	}
}

func TestScheduleSkippedOutsideSyncWindow(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)

	h := time.Now().UTC().Hour()
	outside := fmt.Sprintf("%02d:00-%02d:00", (h+2)%24, (h+3)%24)
	repo.MirrorSyncWindow = outside
	_ = e.store.Update(repo)

	_, err := e.uc.syncMirrorByID(context.Background(), syncRequest{RepoID: repo.ID, Trigger: domain.MirrorTriggerSchedule})
	if !errors.Is(err, ErrMirrorSyncSkipped) || e.git.fetches != 0 {
		t.Fatalf("err=%v fetches=%d", err, e.git.fetches)
	}

	if err := pull(e, repo.ID); err != nil {
		t.Fatal(err)
	}
}

func TestPushDebounceCoalesces(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPush)

	for range 6 {
		e.uc.SchedulePushMirror(repo.ID)
		time.Sleep(5 * time.Millisecond)
	}

	e.uc.WaitMirrorIdle()

	e.git.mu.Lock()
	defer e.git.mu.Unlock()

	if e.git.pushes != 1 {
		t.Fatalf("pushes=%d, want 1", e.git.pushes)
	}
}

func TestSyncBatchBoundedConcurrency(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	cfg := e.uc.mirrorConfig()
	cfg.Workers = 3
	e.uc.SetMirrorConfig(cfg)

	var ids []uint
	for range 9 {
		ids = append(ids, e.addMirror(domain.MirrorPull).ID)
	}

	var cur, peak atomic.Int32

	e.git.fetchFn = func(context.Context, string, string, *domain.MirrorOptions) error {
		n := cur.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}

		time.Sleep(20 * time.Millisecond)
		cur.Add(-1)

		return nil
	}

	e.uc.syncBatch(context.Background(), ids, domain.MirrorTriggerSchedule, 0)

	if p := peak.Load(); p < 2 || p > 3 {
		t.Fatalf("peak concurrency=%d, want 2..3", p)
	}

	if e.git.fetches != 9 {
		t.Fatalf("fetches=%d", e.git.fetches)
	}
}

func TestMirrorBackoffAndDue(t *testing.T) {
	t.Parallel()

	for fails, want := range map[int]time.Duration{1: 5 * time.Minute, 2: 15 * time.Minute, 3: time.Hour, 4: 6 * time.Hour, 20: 6 * time.Hour} {
		if got := mirrorBackoff(fails); got != want {
			t.Fatalf("backoff(%d)=%s want %s", fails, got, want)
		}
	}

	now := time.Now()
	past, future := now.Add(-time.Minute), now.Add(time.Hour)

	if mirrorDue(&domain.Repository{
		MirrorDirection: domain.MirrorPull,
		MirrorURL:       "https://git.example.com/a.git",
		MirrorEnabled:   false},
		time.Hour,
		now,
	) {
		t.Fatal("paused must not be due")
	}

	if !mirrorDue(&domain.Repository{
		MirrorDirection:  domain.MirrorPull,
		MirrorURL:        "https://git.example.com/a.git",
		MirrorEnabled:    true,
		MirrorNextSyncAt: &past,
	},
		time.Hour,
		now,
	) {
		t.Fatal("past next_sync must be due")
	}

	if mirrorDue(&domain.Repository{
		MirrorDirection:  domain.MirrorPull,
		MirrorURL:        "https://git.example.com/a.git",
		MirrorEnabled:    true,
		MirrorNextSyncAt: &future,
	},
		time.Hour,
		now,
	) {
		t.Fatal("future next_sync must not be due")
	}
}

func TestSyncWindow(t *testing.T) {
	t.Parallel()

	at := func(h int) time.Time {
		return time.Date(2026, 1, 1, h, 30, 0, 0, time.UTC)
	}

	cases := []struct {
		window string
		hour   int
		in     bool
	}{
		{"", 3, true},
		{"01:00-05:00", 3, true},
		{"01:00-05:00", 6, false},
		{"22:00-04:00", 23, true},
		{"22:00-04:00", 2, true},
		{"22:00-04:00", 12, false},
	}

	for _, c := range cases {
		if got := inSyncWindow(c.window, at(c.hour)); got != c.in {
			t.Fatalf("window=%q hour=%d got=%v", c.window, c.hour, got)
		}
	}
}

func TestRateLimiterAllowsOncePerWindow(t *testing.T) {
	t.Parallel()

	l := newRateLimiter(15 * time.Second)
	now := time.Now()

	if !l.allow(1, now) || l.allow(1, now.Add(time.Second)) {
		t.Fatal("second call within window must be denied")
	}

	if !l.allow(2, now) {
		t.Fatal("other repo has its own bucket")
	}

	if !l.allow(1, now.Add(time.Minute)) {
		t.Fatal("must allow after refill")
	}
}

func TestConfigurePauseKeepsConfigAndClearRemoves(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)

	base := ConfigureMirrorInput{
		ViewerID:  1,
		Name:      "app",
		Direction: domain.MirrorPull,
		URL:       repo.MirrorURL,
	}

	paused := base
	paused.Enabled = false

	item, err := e.uc.ConfigureMirror(context.Background(), paused)
	if err != nil {
		t.Fatal(err)
	}

	cur := e.store.get(repo.ID)
	if cur.MirrorEnabled || cur.MirrorStatus != domain.MirrorStatusPaused ||
		cur.MirrorURL == "" || cur.MirrorPassword == "" || cur.MirrorDirection != domain.MirrorPull {
		t.Fatalf("pause must keep config: %+v", cur)
	}

	if item.MirrorEnabled == nil || *item.MirrorEnabled || !item.MirrorHasPassword {
		t.Fatalf("item=%+v", item)
	}

	clr := base
	clr.Clear = true

	if _, err := e.uc.ConfigureMirror(context.Background(), clr); err != nil {
		t.Fatal(err)
	}

	cur = e.store.get(repo.ID)
	if cur.MirrorDirection != domain.MirrorNone || cur.MirrorURL != "" || cur.MirrorPassword != "" {
		t.Fatalf("clear must remove config: %+v", cur)
	}
}

func TestConfigureStoresCredsAndNeverReturnsSecrets(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.store.add(domain.Repository{
		Name:          "app",
		RepoPath:      "app.git",
		OwnerKind:     domain.OwnerKindUser,
		OwnerID:       1,
		DefaultBranch: "main",
	})

	pw := "hunter2-secret"
	item, err := e.uc.ConfigureMirror(context.Background(), ConfigureMirrorInput{
		ViewerID:  1,
		Name:      "app",
		Direction: domain.MirrorPush,
		Enabled:   true,
		URL:       "https://git.example.com/org/app.git",
		Username:  new("bot"),
		Password:  &pw,
	})
	if err != nil {
		t.Fatal(err)
	}

	stored := e.store.get(repo.ID)
	if stored.MirrorPassword != pw {
		t.Fatalf("password must be stored: %q", stored.MirrorPassword)
	}

	if !item.MirrorHasPassword || strings.Contains(fmt.Sprintf("%+v", *item), pw) {
		t.Fatalf("response must not echo the password")
	}

	if _, err := e.uc.ConfigureMirror(context.Background(), ConfigureMirrorInput{
		ViewerID:  1,
		Name:      "app",
		Direction: domain.MirrorPush,
		Enabled:   true,
		URL:       "file:///etc",
	}); !errors.Is(err, ErrInvalidMirrorURL) {
		t.Fatalf("bad url: %v", err)
	}
}

func TestCreateMirrorCloneFailureCleansUp(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	e.git.cloneFn = func(context.Context, string, string, *domain.MirrorOptions) (string, error) {
		return "", errors.New("remote: repository not found")
	}

	_, err := e.uc.Create(context.Background(), CreateRepositoryInput{
		ViewerID:        1,
		Name:            "clone-me",
		MirrorDirection: domain.MirrorPull,
		MirrorURL:       "https://git.example.com/org/x.git",
	})
	if err == nil {
		t.Fatal("expected clone failure")
	}

	if len(e.store.repos) != 0 {
		t.Fatalf("db row must be removed, have %d", len(e.store.repos))
	}

	if len(e.git.removed) == 0 {
		t.Fatal("on-disk repo must be removed")
	}
}

func TestMirrorHealthAndHistory(t *testing.T) {
	t.Parallel()

	e := newMirrorEnv()
	repo := e.addMirror(domain.MirrorPull)
	e.git.fetchFn = func(context.Context, string, string, *domain.MirrorOptions) error { return errors.New("boom") }

	_ = pull(e, repo.ID)

	h, err := e.uc.MirrorHealth(context.Background(), time.Hour)
	if err != nil || h == nil {
		t.Fatalf("health=%v err=%v", h, err)
	}

	hist, err := e.uc.MirrorHistory(context.Background(), ResolveRepositoryInput{Name: "app", ViewerID: 1}, 10)
	if err != nil || len(hist) == 0 || hist[0].Status != domain.MirrorEventFailed {
		t.Fatalf("history=%+v err=%v", hist, err)
	}
}
