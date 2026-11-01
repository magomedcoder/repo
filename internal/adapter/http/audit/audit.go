package audit

import (
	"log/slog"
	"net/http"
	"path"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
)

type Logger struct {
	store domain.AccessLogStore
}

func New(store domain.AccessLogStore) *Logger {
	if store == nil {
		return nil
	}

	return &Logger{
		store: store,
	}
}

func (l *Logger) Visit(r *http.Request, user *domain.User) {
	l.VisitPath(r, user, requestPath(r))
}

func (l *Logger) VisitPath(r *http.Request, user *domain.User, pathValue string) {
	if pathValue == "" {
		pathValue = "/"
	}

	l.record(r, domain.AccessVisit, user, "", "", pathValue)
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
	l.record(r, domain.AccessFailedAuth, nil, "", "", truncate(login, 128))
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

	go func() {
		if err := l.store.Create(ev); err != nil {
			slog.Warn("access_log", "action", action, "err", err)
		}
	}()
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
