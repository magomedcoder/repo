package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
)

const (
	securityQueueSize   = 4096
	visitQueueSize      = 512
	securityEnqueueWait = 500 * time.Millisecond
	accessLogRetention  = 90 * 24 * time.Hour
	accessVisitThrottle = time.Minute
	accessPurgeEveryN   = 128
	maxVisitPathLen     = 512
	maxVisitKeys        = 10000
	visitBurstPerIP     = 60
)

type Logger struct {
	store    domain.AccessLogStore
	security chan *domain.AccessLog
	visits   chan *domain.AccessLog

	visitMu   sync.Mutex
	visitLast map[string]time.Time
	visitIP   map[string]visitWindow

	writes atomic.Uint64
}

type visitWindow struct {
	start time.Time
	n     int
}

func New(store domain.AccessLogStore) *Logger {
	if store == nil {
		return nil
	}

	l := &Logger{
		store:     store,
		security:  make(chan *domain.AccessLog, securityQueueSize),
		visits:    make(chan *domain.AccessLog, visitQueueSize),
		visitLast: make(map[string]time.Time),
		visitIP:   make(map[string]visitWindow),
	}
	go l.worker()
	return l
}

func (l *Logger) worker() {
	for {
		select {
		case ev := <-l.security:
			l.write(ev)
			continue
		default:
		}

		select {
		case ev := <-l.security:
			l.write(ev)
		case ev := <-l.visits:
			l.write(ev)
		}
	}
}

func (l *Logger) write(ev *domain.AccessLog) {
	if err := l.store.Create(ev); err != nil {
		slog.Warn("access_log", "action", ev.Action, "err", err)
	}
	n := l.writes.Add(1)
	if n%accessPurgeEveryN == 0 {
		cutoff := time.Now().Add(-accessLogRetention)
		if err := l.store.DeleteOlderThan(cutoff); err != nil {
			slog.Warn("access_log_purge", "err", err)
		}
	}
}

func validVisitPath(p string) bool {
	if p == "" || len(p) > maxVisitPathLen || p[0] != '/' {
		return false
	}

	if p == "/api" || strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/assets/") {
		return false
	}

	if strings.Contains(p, "..") || strings.Contains(p, "//") || strings.Contains(p, "\\") {
		return false
	}

	if strings.ContainsAny(p, "?#%") {
		return false
	}

	for i := range len(p) {
		if p[i] < 0x20 || p[i] == 0x7f {
			return false
		}
	}

	return true
}

func (l *Logger) Visit(r *http.Request, user *domain.User) {
	l.VisitPath(r, user, requestPath(r))
}

func (l *Logger) VisitPath(r *http.Request, user *domain.User, pathValue string) {
	if l == nil {
		return
	}

	if pathValue == "" {
		pathValue = "/"
	}

	if !validVisitPath(pathValue) {
		return
	}

	if !l.allowVisit(r, pathValue) {
		return
	}

	l.record(r, domain.AccessVisit, user, "", "", pathValue)
}

func (l *Logger) allowVisit(r *http.Request, pathValue string) bool {
	ip := middleware.ClientIP(r)
	key := ip + "\n" + pathValue
	now := time.Now()

	l.visitMu.Lock()
	defer l.visitMu.Unlock()

	if last, ok := l.visitLast[key]; ok && now.Sub(last) < accessVisitThrottle {
		return false
	}

	w := l.visitIP[ip]
	if now.Sub(w.start) > accessVisitThrottle {
		w = visitWindow{start: now}
	}

	if w.n >= visitBurstPerIP {
		l.visitIP[ip] = w
		return false
	}

	w.n++
	l.visitIP[ip] = w
	l.visitLast[key] = now

	if len(l.visitLast) > maxVisitKeys {
		for k, t := range l.visitLast {
			if now.Sub(t) > accessVisitThrottle {
				delete(l.visitLast, k)
			}
		}
		if len(l.visitLast) > maxVisitKeys {
			l.visitLast = make(map[string]time.Time)
		}
	}

	if len(l.visitIP) > maxVisitKeys {
		for k, v := range l.visitIP {
			if now.Sub(v.start) > accessVisitThrottle {
				delete(l.visitIP, k)
			}
		}

		if len(l.visitIP) > maxVisitKeys {
			l.visitIP = make(map[string]visitWindow)
		}
	}

	return true
}

func (l *Logger) Clone(r *http.Request, repoPath string, user *domain.User) {
	l.record(r, domain.AccessClone, user, repoPath, "", "")
}

func (l *Logger) Push(r *http.Request, repoPath string, user *domain.User) {
	l.record(r, domain.AccessPush, user, repoPath, "", "")
}

func (l *Logger) Download(r *http.Request, repoPath, ref, filePath string, user *domain.User) {
	l.record(r, domain.AccessDownload, user, repoPath, ref, filePath)
}

func (l *Logger) Login(r *http.Request, user *domain.User) {
	l.record(r, domain.AccessLogin, user, "", "", "")
}

func (l *Logger) Logout(r *http.Request, user *domain.User) {
	l.record(r, domain.AccessLogout, user, "", "", "")
}

func (l *Logger) Register(r *http.Request, user *domain.User) {
	l.record(r, domain.AccessRegister, user, "", "", "")
}

func (l *Logger) FailedAuth(r *http.Request, login string) {
	l.record(r, domain.AccessFailedAuth, nil, "", "", loginFingerprint(login))
}

func (l *Logger) TokenCreate(r *http.Request, user *domain.User, tokenName string) {
	l.record(r, domain.AccessTokenCreate, user, "", "", tokenName)
}

func (l *Logger) TokenRevoke(r *http.Request, user *domain.User, tokenID string) {
	l.record(r, domain.AccessTokenRevoke, user, "", "", tokenID)
}

func (l *Logger) RepoCreate(r *http.Request, user *domain.User, repoPath string) {
	l.record(r, domain.AccessRepoCreate, user, repoPath, "", "")
}

func (l *Logger) RepoDelete(r *http.Request, user *domain.User, repoPath string) {
	l.record(r, domain.AccessRepoDelete, user, repoPath, "", "")
}

func (l *Logger) RepoMove(r *http.Request, user *domain.User, fromPath, toPath string) {
	l.record(r, domain.AccessRepoMove, user, toPath, "", fromPath)
}

func (l *Logger) AdminUserCreate(r *http.Request, actor *domain.User, targetUsername string) {
	l.record(r, domain.AccessAdminUserCreate, actor, "", "", targetUsername)
}

func (l *Logger) AdminUserUpdate(r *http.Request, actor *domain.User, targetID string) {
	l.record(r, domain.AccessAdminUserUpdate, actor, "", "", targetID)
}

func (l *Logger) AdminUserDelete(r *http.Request, actor *domain.User, targetID string) {
	l.record(r, domain.AccessAdminUserDelete, actor, "", "", targetID)
}

func (l *Logger) record(r *http.Request, action string, user *domain.User, repo, ref, pathValue string) {
	if l == nil || l.store == nil || r == nil {
		return
	}

	ev := &domain.AccessLog{
		Action:    action,
		IP:        middleware.ClientIP(r),
		UserAgent: truncate(r.UserAgent(), 512),
		Path:      truncate(pathValue, 1024),
		Repo:      truncate(repo, 512),
		Ref:       truncate(ref, 255),
	}
	if user != nil {
		id := user.ID
		ev.UserID = &id
		ev.Username = user.Username
	}

	if action == domain.AccessVisit {
		select {
		case l.visits <- ev:
		default:
			slog.Debug("access_log_visit_dropped")
		}
		return
	}

	select {
	case l.security <- ev:
		return
	default:
	}

	t := time.NewTimer(securityEnqueueWait)
	defer t.Stop()
	select {
	case l.security <- ev:
	case <-t.C:
		slog.Error("access_log_security_dropped", "action", action)
	}
}

func loginFingerprint(login string) string {
	login = strings.TrimSpace(strings.ToLower(login))
	if login == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(login))

	return hex.EncodeToString(sum[:8])
}

func IsPageVisit(r *http.Request) bool {
	if r == nil || r.Method != http.MethodGet {
		return false
	}

	p := r.URL.Path
	if p == "/api" || strings.HasPrefix(p, "/api/") {
		return false
	}

	if strings.HasPrefix(p, "/assets/") {
		return false
	}

	ext := strings.ToLower(path.Ext(p))
	switch ext {
	case "", ".html":
		return true
	default:
		return false
	}
}

func requestPath(r *http.Request) string {
	if r.URL == nil {
		return "/"
	}

	p := r.URL.Path
	if p == "" {
		return "/"
	}

	return p
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}

	return s[:n]
}

func RepoKey(folderPath, name string) string {
	folderPath = strings.Trim(folderPath, "/")
	name = strings.TrimSpace(name)
	if folderPath == "" {
		return name
	}

	if name == "" {
		return folderPath
	}

	return folderPath + "/" + name
}
