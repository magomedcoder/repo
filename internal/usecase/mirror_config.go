package usecase

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v6/config"
	gossh "golang.org/x/crypto/ssh"

	appconfig "magomedcoder.ru/repo/internal/config"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/mirror/metrics"
)

var (
	ErrInvalidMirrorOption = NewError(KindValidation, "invalid_mirror_option")
	ErrInvalidSSHKey       = NewError(KindValidation, "invalid_ssh_key")
)

const (
	minMirrorInterval = time.Minute
	maxMirrorInterval = 30 * 24 * time.Hour
	mirrorHistoryKeep = 100
)

type MirrorConfig struct {
	Workers         int
	PullInterval    time.Duration
	CloneTimeout    time.Duration
	FetchTimeout    time.Duration
	PushTimeout     time.Duration
	ManualSyncEvery time.Duration
	PushDebounce    time.Duration
	SyncOnConfigure bool
}

func DefaultMirrorConfig() MirrorConfig {
	return MirrorConfig{
		Workers:         appconfig.DefaultMirrorWorkers,
		PullInterval:    appconfig.DefaultMirrorPullInterval,
		CloneTimeout:    appconfig.DefaultMirrorCloneTimeout,
		FetchTimeout:    appconfig.DefaultMirrorFetchTimeout,
		PushTimeout:     appconfig.DefaultMirrorPushTimeout,
		ManualSyncEvery: 15 * time.Second,
		PushDebounce:    3 * time.Second,
		SyncOnConfigure: true,
	}
}

func MirrorConfigFromApp(cfg appconfig.Config) MirrorConfig {
	mc := DefaultMirrorConfig()
	mc.Workers = cfg.MirrorWorkers
	mc.PullInterval = cfg.MirrorPullEvery

	if cfg.MirrorCloneTimeoutDur > 0 {
		mc.CloneTimeout = cfg.MirrorCloneTimeoutDur
	}

	if cfg.MirrorFetchTimeoutDur > 0 {
		mc.FetchTimeout = cfg.MirrorFetchTimeoutDur
	}

	if cfg.MirrorPushTimeoutDur > 0 {
		mc.PushTimeout = cfg.MirrorPushTimeoutDur
	}

	return mc.normalized()
}

func (c MirrorConfig) normalized() MirrorConfig {
	d := DefaultMirrorConfig()
	if c.Workers <= 0 {
		c.Workers = d.Workers
	}

	if c.CloneTimeout <= 0 {
		c.CloneTimeout = d.CloneTimeout
	}

	if c.FetchTimeout <= 0 {
		c.FetchTimeout = d.FetchTimeout
	}

	if c.PushTimeout <= 0 {
		c.PushTimeout = d.PushTimeout
	}

	if c.ManualSyncEvery <= 0 {
		c.ManualSyncEvery = d.ManualSyncEvery
	}

	if c.PushDebounce <= 0 {
		c.PushDebounce = d.PushDebounce
	}

	return c
}

type mirrorRuntime struct {
	cfg     MirrorConfig
	metrics *metrics.Metrics
	events  domain.MirrorSyncStore
	targets domain.MirrorTargetStore
	groups  domain.MirrorGroupStore
	limiter *rateLimiter
	rootCtx context.Context
	slots   chan struct{}

	wg       sync.WaitGroup
	mu       sync.Mutex
	locks    map[uint]*sync.Mutex
	debounce map[uint]*time.Timer
	pending  map[uint]bool
}

func newSyncSlots(n int) chan struct{} {
	if n <= 0 {
		n = 1
	}

	ch := make(chan struct{}, n)
	for range n {
		ch <- struct{}{}
	}

	return ch
}

func newMirrorRuntime() *mirrorRuntime {
	cfg := DefaultMirrorConfig()

	return &mirrorRuntime{
		cfg:      cfg,
		metrics:  metrics.New(),
		limiter:  newRateLimiter(cfg.ManualSyncEvery),
		slots:    newSyncSlots(cfg.Workers),
		locks:    make(map[uint]*sync.Mutex),
		debounce: make(map[uint]*time.Timer),
		pending:  make(map[uint]bool),
	}
}

func (uc *RepositoryUseCase) SetMirrorConfig(cfg MirrorConfig) {
	cfg = cfg.normalized()
	uc.mirror.mu.Lock()
	oldWorkers := uc.mirror.cfg.Workers
	uc.mirror.cfg = cfg
	uc.mirror.limiter = newRateLimiter(cfg.ManualSyncEvery)
	if cfg.Workers != oldWorkers || uc.mirror.slots == nil {
		uc.mirror.slots = newSyncSlots(cfg.Workers)
	}
	uc.mirror.mu.Unlock()
}

func (uc *RepositoryUseCase) SetMirrorRootContext(ctx context.Context) {
	uc.mirror.mu.Lock()
	uc.mirror.rootCtx = ctx
	uc.mirror.mu.Unlock()
}

func (uc *RepositoryUseCase) mirrorRootContext() context.Context {
	uc.mirror.mu.Lock()
	defer uc.mirror.mu.Unlock()

	if uc.mirror.rootCtx != nil {
		return uc.mirror.rootCtx
	}

	return context.Background()
}

func (uc *RepositoryUseCase) acquireSyncSlot(ctx context.Context) (chan struct{}, error) {
	uc.mirror.mu.Lock()
	slots := uc.mirror.slots
	uc.mirror.mu.Unlock()

	if slots == nil {
		return nil, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-slots:
		return slots, nil
	}
}

func (uc *RepositoryUseCase) releaseSyncSlot(slots chan struct{}) {
	if slots == nil {
		return
	}

	select {
	case slots <- struct{}{}:
	default:
	}
}

func (uc *RepositoryUseCase) SetMirrorSyncStore(s domain.MirrorSyncStore) {
	uc.mirror.events = s
}

func (uc *RepositoryUseCase) SetMirrorTargetStore(s domain.MirrorTargetStore) {
	uc.mirror.targets = s
}

func (uc *RepositoryUseCase) SetMirrorGroupStore(s domain.MirrorGroupStore) {
	uc.mirror.groups = s
}

func (uc *RepositoryUseCase) SetMetrics(m *metrics.Metrics) {
	if m != nil {
		uc.mirror.metrics = m
	}
}

func (uc *RepositoryUseCase) MirrorMetrics() *metrics.Metrics {
	return uc.mirror.metrics
}

func (uc *RepositoryUseCase) mirrorConfig() MirrorConfig {
	uc.mirror.mu.Lock()
	defer uc.mirror.mu.Unlock()

	return uc.mirror.cfg
}

func (uc *RepositoryUseCase) WaitMirrorIdle() {
	uc.mirror.wg.Wait()
}

func (uc *RepositoryUseCase) StopMirrorDebouncers() {
	uc.mirror.mu.Lock()
	defer uc.mirror.mu.Unlock()

	for id, t := range uc.mirror.debounce {
		t.Stop()
		delete(uc.mirror.debounce, id)
	}
}

func (uc *RepositoryUseCase) goMirror(fn func()) {

	uc.mirror.wg.Go(func() {
		fn()
	})
}

func (uc *RepositoryUseCase) sealSecret(plain string) (string, error) {
	return plain, nil
}

func (uc *RepositoryUseCase) openSecret(stored string) (string, error) {
	return stored, nil
}

type mirrorURLPolicy struct{}

func (uc *RepositoryUseCase) urlPolicy() mirrorURLPolicy {
	return mirrorURLPolicy{}
}

func (uc *RepositoryUseCase) validateMirrorURL(raw string) error {
	return uc.urlPolicy().validate(raw)
}

var scpLikeURL = regexp.MustCompile(`^[A-Za-z0-9._-]+@(\[[0-9a-fA-F:]+\]|[A-Za-z0-9.-]+):[^\s]+$`)

func parseMirrorHost(raw string) (scheme, host string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\r\n") {
		return "", "", false
	}

	if !strings.Contains(raw, "://") {
		if !scpLikeURL.MatchString(raw) {
			return "", "", false
		}

		after := raw[strings.Index(raw, "@")+1:]
		if strings.HasPrefix(after, "[") {
			host = after[1:strings.Index(after, "]")]
		} else {
			host = after[:strings.Index(after, ":")]
		}

		return "ssh", strings.ToLower(host), true
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", "", false
	}

	scheme = strings.ToLower(u.Scheme)
	switch scheme {
	case "https", "http":
		if u.User != nil {
			return "", "", false
		}
	case "ssh":
	default:
		return "", "", false
	}

	return scheme, strings.ToLower(u.Hostname()), true
}

func (p mirrorURLPolicy) validate(raw string) error {
	_, host, ok := parseMirrorHost(raw)
	if !ok || host == "" || p.hostBlocked(host) {
		return ErrInvalidMirrorURL
	}

	return nil
}

func cloudMetadataHost(host string) bool {
	return host == "metadata.google.internal" || host == "169.254.169.254" || host == "fd00:ec2::254"
}

func internalIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func (p mirrorURLPolicy) hostBlocked(host string) bool {
	if cloudMetadataHost(host) {
		return true
	}

	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return internalIP(ip)
	}

	return false
}

func (p mirrorURLPolicy) parsePinnedHosts(raw string) (map[string]string, error) {
	out := map[string]string{}

	for _, part := range splitList(raw) {
		host, ipStr, ok := strings.Cut(part, "=")
		host = strings.ToLower(strings.TrimSpace(host))
		ip := net.ParseIP(strings.TrimSpace(ipStr))

		if !ok || host == "" || ip == nil {
			return nil, ErrInvalidMirrorOption
		}

		if internalIP(ip) {
			return nil, ErrInvalidMirrorURL
		}

		out[host] = ip.String()
	}

	return out, nil
}

func splitList(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';'
	})
	out := make([]string, 0, len(fields))

	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}

	return out
}

func parseRefspecs(raw string) []string {
	return splitList(raw)
}

func normalizeRefspecs(raw string) (string, error) {
	list := parseRefspecs(raw)
	for _, s := range list {
		spec := strings.TrimPrefix(s, "+")
		if !strings.HasPrefix(spec, "refs/") || strings.Contains(spec, "..") {
			return "", ErrInvalidMirrorOption
		}

		full := spec
		if !strings.Contains(full, ":") {
			full = full + ":" + full
		}

		if err := config.RefSpec(full).Validate(); err != nil {
			return "", ErrInvalidMirrorOption
		}
	}

	return strings.Join(list, ","), nil
}

func normalizeInterval(raw string) (string, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "":
		return "", nil
	case "0", "off", "disabled", "never":
		return "0", nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d < minMirrorInterval || d > maxMirrorInterval {
		return "", ErrInvalidMirrorOption
	}

	return d.String(), nil
}

func effectiveMirrorInterval(repo *domain.Repository, global time.Duration) time.Duration {
	switch repo.MirrorInterval {
	case "":
		return global
	case "0":
		return 0
	}

	d, err := time.ParseDuration(repo.MirrorInterval)
	if err != nil || d <= 0 {
		return global
	}

	return d
}

var syncWindowPattern = regexp.MustCompile(`^(\d{2}):(\d{2})-(\d{2}):(\d{2})$`)

func parseSyncWindow(raw string) (start, end int, ok bool) {
	m := syncWindowPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if m == nil {
		return 0, 0, false
	}

	var v [4]int
	for i := range v {
		fmt.Sscanf(m[i+1], "%d", &v[i])
	}

	if v[0] > 23 || v[2] > 23 || v[1] > 59 || v[3] > 59 {
		return 0, 0, false
	}

	start, end = v[0]*60+v[1], v[2]*60+v[3]

	return start, end, start != end
}

func normalizeSyncWindow(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	if _, _, ok := parseSyncWindow(raw); !ok {
		return "", ErrInvalidMirrorOption
	}

	return raw, nil
}

func inSyncWindow(window string, now time.Time) bool {
	if strings.TrimSpace(window) == "" {
		return true
	}

	start, end, ok := parseSyncWindow(window)
	if !ok {
		return true
	}

	now = now.UTC()
	m := now.Hour()*60 + now.Minute()
	if start < end {
		return m >= start && m < end
	}

	return m >= start || m < end
}

func normalizeConflictPolicy(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", domain.MirrorConflictForce:
		return domain.MirrorConflictForce, nil
	case domain.MirrorConflictReject:
		return domain.MirrorConflictReject, nil
	default:
		return "", ErrInvalidMirrorOption
	}
}

func mirrorBackoff(fails int) time.Duration {
	steps := []time.Duration{5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour}
	if fails < 1 {
		return 0
	}

	if fails > len(steps) {
		fails = len(steps)
	}

	return steps[fails-1]
}

func validateSSHKey(key, passphrase string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}

	_, err := gossh.ParsePrivateKey([]byte(key))
	if _, missing := err.(*gossh.PassphraseMissingError); missing {
		_, err = gossh.ParsePrivateKeyWithPassphrase([]byte(key), []byte(passphrase))
	}

	if err != nil {
		return ErrInvalidSSHKey
	}

	return nil
}

type bucket struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	mu    sync.Mutex
	every time.Duration
	burst float64
	m     map[uint]*bucket
}

func newRateLimiter(every time.Duration) *rateLimiter {
	return &rateLimiter{every: every, burst: 1, m: make(map[uint]*bucket)}
}

const rateLimiterMaxBuckets = 2048

func (l *rateLimiter) allow(id uint, now time.Time) bool {
	if l == nil || l.every <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.gc(now)

	b, ok := l.m[id]
	if !ok {
		l.m[id] = &bucket{
			tokens: l.burst - 1,
			last:   now,
		}

		return true
	}

	b.tokens += now.Sub(b.last).Seconds() / l.every.Seconds()
	if b.tokens > l.burst {
		b.tokens = l.burst
	}

	b.last = now
	if b.tokens < 1 {
		return false
	}

	b.tokens--

	return true
}

func (l *rateLimiter) gc(now time.Time) {
	idle := 10 * l.every
	if idle <= 0 {
		idle = time.Minute
	}

	for id, b := range l.m {
		if now.Sub(b.last) > idle {
			delete(l.m, id)
		}
	}

	for len(l.m) > rateLimiterMaxBuckets {
		for id := range l.m {
			delete(l.m, id)
			break
		}
	}
}
