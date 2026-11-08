package usecase

import (
	"context"
	"errors"
	"net"
	"slices"
	"sync"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/secretbox"
	"magomedcoder.ru/repo/pkg/ssrf"
)

type fakeStore struct {
	domain.RepositoryStore
	mu     sync.Mutex
	repos  map[uint]*domain.Repository
	next   uint
	purged []uint
}

func newFakeStore() *fakeStore {
	return &fakeStore{repos: map[uint]*domain.Repository{}}
}

func (s *fakeStore) add(r domain.Repository) *domain.Repository {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	r.ID = s.next
	cp := r
	s.repos[r.ID] = &cp

	return &cp
}

func (s *fakeStore) get(id uint) domain.Repository {
	s.mu.Lock()
	defer s.mu.Unlock()

	return *s.repos[id]
}

func (s *fakeStore) Create(r *domain.Repository) error {
	s.add(*r)
	s.mu.Lock()
	r.ID = s.next
	s.mu.Unlock()

	return nil
}

func (s *fakeStore) Update(r *domain.Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *r
	s.repos[r.ID] = &cp

	return nil
}

func (s *fakeStore) Delete(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.repos, id)

	return nil
}

func (s *fakeStore) FindByID(_ context.Context, id uint) (*domain.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.repos[id]
	if !ok {
		return nil, errors.New("not found")
	}

	cp := *r

	return &cp, nil
}

func (s *fakeStore) FindByRepoPath(_ context.Context, p string) (*domain.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.repos {
		if r.RepoPath == p {
			cp := *r

			return &cp, nil
		}
	}

	return nil, errors.New("not found")
}

func (s *fakeStore) ExistsByName(string) (bool, error) {
	return false, nil
}

func (s *fakeStore) CountByOwnerID(uint) (int64, error) {
	return 0, nil
}

func (s *fakeStore) UpdateMirrorState(_ context.Context, r *domain.Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur := s.repos[r.ID]
	if cur == nil {
		return errors.New("not found")
	}

	cur.DefaultBranch = r.DefaultBranch
	cur.MirrorLastSyncAt = r.MirrorLastSyncAt
	cur.MirrorLastError = r.MirrorLastError
	cur.MirrorStatus = r.MirrorStatus
	cur.MirrorFailCount = r.MirrorFailCount
	cur.MirrorNextSyncAt = r.MirrorNextSyncAt
	cur.MirrorLastProgress = r.MirrorLastProgress
	cur.MirrorRemoteHEAD = r.MirrorRemoteHEAD

	return nil
}

func (s *fakeStore) ListMirrors(_ context.Context, f domain.MirrorListFilter) ([]domain.Repository, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []domain.Repository

	for id := uint(1); id <= s.next; id++ {
		r, ok := s.repos[id]
		if !ok || r.MirrorDirection == "" || (f.Direction != "" && r.MirrorDirection != f.Direction) {
			continue
		}

		if f.Status != "" && r.MirrorStatus != f.Status {
			continue
		}

		out = append(out, *r)
	}

	return out, nil
}

func (s *fakeStore) ResetStaleMirrorSyncing(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var n int64
	for _, r := range s.repos {
		if r.MirrorStatus == domain.MirrorStatusSyncing {
			r.MirrorStatus = domain.MirrorStatusIdle
			n++
		}
	}

	return n, nil
}

func (s *fakeStore) ListMirrorSchedules(_ context.Context, f domain.MirrorScheduleFilter) ([]domain.MirrorSchedule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	limit, offset := f.Limit, f.Offset
	if limit <= 0 {
		limit = 100
	}

	var out []domain.MirrorSchedule
	for id := uint(1); id <= s.next; id++ {
		r, ok := s.repos[id]
		if !ok || r.MirrorDirection == "" || r.MirrorURL == "" {
			continue
		}

		if f.DueBefore != nil {
			if !r.MirrorEnabled {
				continue
			}
			if r.MirrorNextSyncAt != nil && r.MirrorNextSyncAt.After(*f.DueBefore) {
				continue
			}
		}

		out = append(out, domain.MirrorSchedule{
			ID:         r.ID,
			Direction:  r.MirrorDirection,
			URL:        r.MirrorURL,
			Enabled:    r.MirrorEnabled,
			Interval:   r.MirrorInterval,
			SyncWindow: r.MirrorSyncWindow,
			NextSyncAt: r.MirrorNextSyncAt,
			LastSyncAt: r.MirrorLastSyncAt,
		})
	}

	if offset > len(out) {
		return nil, nil
	}

	out = out[offset:]
	if len(out) > limit {
		out = out[:limit]
	}

	return out, nil
}

func (s *fakeStore) ListAdmin(_ context.Context, filter domain.AdminRepoFilter) ([]domain.Repository, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Repository, 0, len(s.repos))
	for _, r := range s.repos {
		out = append(out, *r)
	}

	return out, len(out), nil
}

func (s *fakeStore) CountAll(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int64(len(s.repos)), nil
}

func (s *fakeStore) CountMirrorsInError(_ context.Context, since time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var n int64

	for _, r := range s.repos {
		if r.MirrorDirection != "" && r.MirrorEnabled && r.MirrorStatus == domain.MirrorStatusError &&
			(r.MirrorLastSyncAt == nil || r.MirrorLastSyncAt.Before(since)) {
			n++
		}
	}

	return n, nil
}

type fakeEvents struct {
	mu     sync.Mutex
	events []domain.MirrorSyncEvent
}

func (e *fakeEvents) Create(_ context.Context, ev *domain.MirrorSyncEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	ev.ID = uint(len(e.events) + 1)
	e.events = append(e.events, *ev)

	return nil
}

func (e *fakeEvents) Update(_ context.Context, ev *domain.MirrorSyncEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.events[ev.ID-1] = *ev

	return nil
}

func (e *fakeEvents) ListByRepo(_ context.Context, repoID uint, limit int) ([]domain.MirrorSyncEvent, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var out []domain.MirrorSyncEvent

	for _, v := range slices.Backward(e.events) {
		if v.RepoID == repoID {
			out = append(out, v)
		}
	}

	return out, nil
}

func (e *fakeEvents) Prune(context.Context, uint, int) error {
	return nil
}

func (e *fakeEvents) DeleteByRepo(_ context.Context, repoID uint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	kept := e.events[:0]
	for _, ev := range e.events {
		if ev.RepoID != repoID {
			kept = append(kept, ev)
		}
	}
	e.events = kept

	return nil
}

func (e *fakeEvents) snapshot() []domain.MirrorSyncEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]domain.MirrorSyncEvent(nil), e.events...)
}

type fakeGit struct {
	domain.GitRepository
	mu       sync.Mutex
	fetchFn  func(ctx context.Context, path, url string, o *domain.MirrorOptions) error
	pushFn   func(ctx context.Context, path, url string, o *domain.MirrorOptions) error
	cloneFn  func(ctx context.Context, path, url string, o *domain.MirrorOptions) (string, error)
	branches []domain.RefInfo
	head     string
	removed  []string
	pushes   int
	fetches  int
}

func (g *fakeGit) GetReposRoot() string {
	return "/tmp/repos"
}

func (g *fakeGit) MirrorFetch(ctx context.Context, path, url string, o *domain.MirrorOptions) error {
	g.mu.Lock()
	g.fetches++
	fn := g.fetchFn
	g.mu.Unlock()

	if fn != nil {
		return fn(ctx, path, url, o)
	}

	return nil
}

func (g *fakeGit) MirrorPush(ctx context.Context, path, url string, o *domain.MirrorOptions) error {
	g.mu.Lock()
	g.pushes++
	fn := g.pushFn
	g.mu.Unlock()

	if fn != nil {
		return fn(ctx, path, url, o)
	}

	return nil
}

func (g *fakeGit) MirrorClone(ctx context.Context, path, url string, o *domain.MirrorOptions) (string, error) {
	if g.cloneFn != nil {
		return g.cloneFn(ctx, path, url, o)
	}

	return "main", nil
}

func (g *fakeGit) MirrorCheck(context.Context, string, *domain.MirrorOptions) (*domain.MirrorRemoteInfo, error) {
	return &domain.MirrorRemoteInfo{
		DefaultBranch: "main",
		Head:          "abc",
		Refs: []domain.RefInfo{
			{
				Name:      "refs/heads/main",
				CommitSHA: "abc",
			},
		},
	}, nil
}

func (g *fakeGit) MirrorDryRunFetch(context.Context, string, string, *domain.MirrorOptions) (*domain.MirrorRemoteInfo, []domain.RefInfo, error) {
	info := &domain.MirrorRemoteInfo{
		DefaultBranch: "main",
		Head:          "abc",
		Refs: []domain.RefInfo{
			{
				Name:      "refs/heads/main",
				CommitSHA: "abc",
			},
		},
	}

	return info, nil, nil
}

func (g *fakeGit) MirrorVerify(context.Context, string, string, string, *domain.MirrorOptions) (*domain.MirrorVerifyResult, error) {
	return &domain.MirrorVerifyResult{
		OK:      true,
		Checked: 1,
	}, nil
}

func (g *fakeGit) ListBranches(string) ([]domain.RefInfo, error) {
	return g.branches, nil
}

func (g *fakeGit) ListTags(string) ([]domain.RefInfo, error) {
	return nil, nil
}

func (g *fakeGit) EnsureDefaultBranch(string, string) error {
	return nil
}

func (g *fakeGit) TipSHA(string) (string, error) {
	return "", nil
}

func (g *fakeGit) WalkTextFiles(context.Context, string) (string, []domain.IndexedTextFile, error) {
	return "", nil, nil
}

func (g *fakeGit) Grep(context.Context, string, string, int) ([]domain.CodeSearchHit, error) {
	return nil, nil
}

func (g *fakeGit) DetectHEADBranch(string) (string, error) {
	return g.head, nil
}

func (g *fakeGit) VerifyCommitSignature(context.Context, string, string) error {
	return nil
}

func (g *fakeGit) LockRepo(string) func() {
	return func() {}
}

func (g *fakeGit) Remove(path string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.removed = append(g.removed, path)

	return nil
}

func (g *fakeGit) InitBare(string, domain.BareInitOptions) error {
	return nil
}

type mirrorEnv struct {
	uc     *RepositoryUseCase
	store  *fakeStore
	git    *fakeGit
	events *fakeEvents
	acts   *memActivity
}

func newMirrorEnv() *mirrorEnv {
	store := newFakeStore()
	git := &fakeGit{branches: []domain.RefInfo{
		{
			Name:      "main",
			CommitSHA: "aaa",
		},
	}, head: "main"}
	users := &memUsers{items: []*domain.User{
		{
			ID:           1,
			Username:     "owner",
			Role:         domain.RoleAdmin,
			PasswordHash: "h:secret",
		},
		{
			ID:       2,
			Username: "dev",
		},
	}, next: 2}
	acts := &memActivity{}
	uc := NewRepositoryUseCase(store, users, git, acts, NewAuthz(nil, nil))
	uc.SetPasswordHasher(memHasher{})
	key, err := secretbox.DeriveKey("0123456789abcdef0123456789abcdef")
	if err != nil {
		panic(err)
	}

	uc.SetMirrorSecretsKey(key)
	uc.SetMirrorURLLookup(func(_ context.Context, host string) ([]net.IP, error) {
		if ssrf.HostBlockedLiteral(host) {
			return nil, ssrf.ErrBlockedHost
		}
		return []net.IP{net.ParseIP("203.0.113.10")}, nil
	})

	cfg := DefaultMirrorConfig()
	cfg.SyncOnConfigure = false
	cfg.PushDebounce = 40 * time.Millisecond
	cfg.ManualSyncEvery = 20 * time.Millisecond
	uc.SetMirrorConfig(cfg)

	events := &fakeEvents{}
	uc.SetMirrorSyncStore(events)

	return &mirrorEnv{
		uc:     uc,
		store:  store,
		git:    git,
		events: events,
		acts:   acts,
	}
}

func (e *mirrorEnv) addMirror(dir string) *domain.Repository {
	sealed, err := e.uc.sealSecret("s3cret-pw")
	if err != nil {
		panic(err)
	}

	return e.store.add(domain.Repository{
		Name:                 "app",
		RepoPath:             "app.git",
		OwnerID:              1,
		DefaultBranch:        "main",
		Path:                 "/tmp/repos/app.git",
		MirrorDirection:      dir,
		MirrorURL:            "https://git.example.com/org/app.git",
		MirrorUsername:       "bot",
		MirrorPassword:       sealed,
		MirrorEnabled:        true,
		MirrorForce:          true,
		MirrorPrune:          true,
		MirrorConflictPolicy: domain.MirrorConflictForce,
		MirrorStatus:         domain.MirrorStatusIdle,
	})
}

func (s *fakeStore) PurgeSoftDeleted(context.Context, time.Time) ([]uint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]uint(nil), s.purged...), nil
}
