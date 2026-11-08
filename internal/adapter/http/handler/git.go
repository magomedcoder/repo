package handler

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
)

const uploadPackMaxBytes = 32 << 20

type pushMirrorScheduler interface {
	SchedulePushMirror(repoID uint)
	ScheduleCodeIndex(repoID uint)
}

type GitHandler struct {
	git             *usecase.GitUseCase
	mirrors         pushMirrorScheduler
	access          *audit.Logger
	receiveMaxBytes int64
}

func NewGitHandler(git *usecase.GitUseCase) *GitHandler {
	return &GitHandler{git: git, receiveMaxBytes: 256 << 20}
}

func (h *GitHandler) SetAccessLogger(log *audit.Logger) {
	h.access = log
}

func (h *GitHandler) SetMirrorScheduler(m pushMirrorScheduler) {
	h.mirrors = m
}

func (h *GitHandler) SetReceiveMaxBytes(n int64) {
	if n > 0 {
		h.receiveMaxBytes = n
	}
}

func IsGitHTTPPath(path string) bool {
	_, _, err := parseGitHTTPPath(path)
	return err == nil
}

func (h *GitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	repoPath, action, err := parseGitHTTPPath(r.URL.Path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if strings.Contains(repoPath, "/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "info/refs":
		h.infoRefs(w, r, repoPath)
	case r.Method == http.MethodPost && action == "git-upload-pack":
		h.servePack(w, r, repoPath, usecase.GitUploadPack)
	case r.Method == http.MethodPost && action == "git-receive-pack":
		h.servePack(w, r, repoPath, usecase.GitReceivePack)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *GitHandler) infoRefs(w http.ResponseWriter, r *http.Request, repoPath string) {
	service, err := usecase.ParseGitService(r.URL.Query().Get("service"))
	if err != nil {
		http.Error(w, "unsupported service", http.StatusForbidden)
		return
	}

	repo, _, err := h.resolveAndAuth(w, r, repoPath, service)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/x-"+string(service)+"-advertisement")
	w.Header().Set("Cache-Control", "no-cache")
	if err := h.git.AdvertiseRefs(r.Context(), repo.Path, service, w); err != nil {
		http.Error(w, "git error", http.StatusInternalServerError)
	}
}

func (h *GitHandler) servePack(w http.ResponseWriter, r *http.Request, repoPath string, service usecase.GitService) {
	repo, viewer, err := h.resolveAndAuth(w, r, repoPath, service)
	if err != nil {
		return
	}

	if !gitPackContentTypeOK(r, service) {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	limit := int64(uploadPackMaxBytes)
	var receive *domain.ReceivePackOptions

	if service == usecase.GitReceivePack {
		unlock := h.git.LockRepo(repo.Path)
		defer unlock()

		limit = h.receiveMaxBytes
		if limit <= 0 {
			limit = 256 << 20
		}
		if repo.ProtectDefaultBranch {
			receive = &domain.ReceivePackOptions{
				ProtectDefaultBranch: true,
				DefaultBranch:        repo.DefaultBranch,
			}
		}
	}

	ctx := r.Context()
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, gitServeTimeout)
		defer cancel()
	}

	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(gitServeTimeout))

	body, err := gitPackBody(w, r, limit)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = body.Close()
	}()

	w.Header().Set("Content-Type", "application/x-"+string(service)+"-result")
	w.Header().Set("Cache-Control", "no-cache")
	tw := &trackingWriter{ResponseWriter: w}
	if err := h.git.ServePack(ctx, repo.Path, service, body, tw, receive); err != nil {
		h.writeServePackError(w, tw, service, err)
		return
	}

	if h.access != nil {
		switch service {
		case usecase.GitUploadPack:
			h.access.Clone(r, repo.RepoPath, viewer)
		case usecase.GitReceivePack:
			h.access.Push(r, repo.RepoPath, viewer)
		}
	}

	if service == usecase.GitReceivePack {
		var actorID uint
		if viewer != nil {
			actorID = viewer.ID
		}
		_ = h.git.TouchActivity(r.Context(), repo, actorID)
		_ = h.git.EnsureDefaultBranch(r.Context(), repo.Path, repo.DefaultBranch)
		if h.mirrors != nil {
			h.mirrors.ScheduleCodeIndex(repo.ID)
			if repo.MirrorDirection == domain.MirrorPush {
				h.mirrors.SchedulePushMirror(repo.ID)
			}
		}
	}
}

func (h *GitHandler) resolveAndAuth(w http.ResponseWriter, r *http.Request, repoPath string, service usecase.GitService) (*domain.Repository, *domain.User, error) {
	viewer, authErr := h.authenticateRequest(r)
	presented := credentialsPresented(r)

	repo, err := h.git.ResolveClonePath(r.Context(), repoPath)
	if err != nil {
		if authErr != nil && presented {
			h.challengeGitAuth(w, r, authErr)
			return nil, nil, authErr
		}

		if !presented && (service == usecase.GitReceivePack || h.git.AnonymousReadDenied()) {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil, nil, usecase.ErrUnauthorized
		}
		http.Error(w, "not found", http.StatusNotFound)

		return nil, nil, err
	}

	needAuth := repo.IsPrivate || service == usecase.GitReceivePack
	if needAuth && authErr != nil {
		if !presented && service == usecase.GitReceivePack {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil, nil, usecase.ErrUnauthorized
		}

		if !presented && h.git.AnonymousReadDenied() {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil, nil, usecase.ErrUnauthorized
		}

		if !presented && repo.IsPrivate {
			http.Error(w, "not found", http.StatusNotFound)
			return nil, nil, usecase.ErrRepoNotFound
		}

		h.challengeGitAuth(w, r, authErr)
		return nil, nil, authErr
	}

	if authErr != nil {
		viewer = nil
	}

	if err := h.git.Authorize(r.Context(), repo, viewer, service); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			if !presented && service == usecase.GitReceivePack {
				w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return nil, nil, err
			}

			if !presented && h.git.AnonymousReadDenied() {
				w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return nil, nil, err
			}

			if !presented && repo.IsPrivate {
				http.Error(w, "not found", http.StatusNotFound)
				return nil, nil, err
			}

			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil, nil, err
		}
		if errors.Is(err, usecase.ErrMirrorReadOnly) {
			http.Error(w, "mirror is read-only", http.StatusForbidden)
			return nil, nil, err
		}
		if errors.Is(err, usecase.ErrRepoForbidden) {
			if service == usecase.GitReceivePack {
				http.Error(w, "forbidden", http.StatusForbidden)
				return nil, nil, err
			}
			http.Error(w, "not found", http.StatusNotFound)

			return nil, nil, err
		}

		http.Error(w, "not found", http.StatusNotFound)
		return nil, nil, err
	}

	return repo, viewer, nil
}

func gitPackContentTypeOK(r *http.Request, service usecase.GitService) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	return strings.EqualFold(mt, "application/x-"+string(service)+"-request")
}

func credentialsPresented(r *http.Request) bool {
	if _, _, ok := r.BasicAuth(); ok {
		return true
	}

	auth := r.Header.Get("Authorization")
	return len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") && strings.TrimSpace(auth[7:]) != ""
}

func (h *GitHandler) challengeGitAuth(w http.ResponseWriter, r *http.Request, authErr error) {
	if h.access != nil {
		login, _, _ := r.BasicAuth()
		h.access.FailedAuth(r, login)
	}

	if errors.Is(authErr, usecase.ErrTooManyAttempts) {
		login, _, _ := r.BasicAuth()
		if ra := h.git.LoginRetryAfter(r.Context(), login, middleware.ClientIPKey(r)); ra > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(ra.Seconds())+1))
		}
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}

	if errors.Is(authErr, usecase.ErrBusy) {
		w.Header().Set("Retry-After", "5")
		http.Error(w, "server busy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func (h *GitHandler) authenticateRequest(r *http.Request) (*domain.User, error) {
	if user, pass, ok := r.BasicAuth(); ok {
		u, err := h.git.Authenticate(r.Context(), middleware.ClientIPKey(r), user, pass)
		if errors.Is(err, usecase.ErrTooManyAttempts) || errors.Is(err, usecase.ErrBusy) {
			return nil, err
		}

		if err == nil && u != nil && strings.HasPrefix(pass, "repo_") && h.access != nil {
			h.access.GitToken(r, u, "")
		}

		return u, err
	}

	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		tok := strings.TrimSpace(auth[7:])
		u, err := h.git.Authenticate(r.Context(), middleware.ClientIPKey(r), "", tok)
		if errors.Is(err, usecase.ErrTooManyAttempts) || errors.Is(err, usecase.ErrBusy) {
			return nil, err
		}

		if err == nil && u != nil && strings.HasPrefix(tok, "repo_") && h.access != nil {
			h.access.GitToken(r, u, "")
		}

		return u, err
	}

	return nil, usecase.ErrUnauthorized
}

const gitServeTimeout = 15 * time.Minute

type trackingWriter struct {
	http.ResponseWriter
	wrote bool
}

func (t *trackingWriter) Write(p []byte) (int, error) {
	if len(p) > 0 {
		t.wrote = true
	}

	return t.ResponseWriter.Write(p)
}

func (t *trackingWriter) Flush() {
	if f, ok := t.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (t *trackingWriter) Unwrap() http.ResponseWriter { return t.ResponseWriter }

func pktLine(s string) string {
	return fmt.Sprintf("%04x%s", len(s)+4, s)
}

func (h *GitHandler) writeServePackError(w http.ResponseWriter, tw *trackingWriter, service usecase.GitService, err error) {
	switch {
	case errors.Is(err, domain.ErrBranchProtected):
		if !tw.wrote {
			_, _ = io.WriteString(w, pktLine("ERR branch is protected\n"))
		}
	case errors.Is(err, domain.ErrBusy):
		if !tw.wrote {
			w.Header().Set("Retry-After", "5")
			http.Error(w, "server busy", http.StatusServiceUnavailable)
		}
	default:
		if !tw.wrote {
			http.Error(w, "git error", http.StatusInternalServerError)
		}
	}
}

type gzipPackBody struct {
	io.Reader
	gz   *gzip.Reader
	body io.Closer
}

func (b *gzipPackBody) Close() error {
	return errors.Join(b.gz.Close(), b.body.Close())
}

func gitPackBody(w http.ResponseWriter, r *http.Request, limit int64) (io.ReadCloser, error) {
	return gitPackBodyLimits(w, r, limit, limit)
}

func gitPackBodyLimits(w http.ResponseWriter, r *http.Request, compressedMax, decompressedMax int64) (io.ReadCloser, error) {
	body := http.MaxBytesReader(w, r.Body, compressedMax)
	if !strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		return body, nil
	}

	gz, err := gzip.NewReader(body)
	if err != nil {
		_ = body.Close()
		return nil, err
	}

	return &gzipPackBody{
		Reader: http.MaxBytesReader(w, gz, decompressedMax),
		gz:     gz,
		body:   body,
	}, nil
}

func parseGitHTTPPath(urlPath string) (repoPath, action string, err error) {
	if strings.Contains(strings.ToLower(urlPath), "%2e") {
		return "", "", errors.New("invalid_repository_path")
	}

	urlPath = strings.Trim(urlPath, "/")
	switch {
	case strings.HasSuffix(urlPath, ".git/info/refs"):
		repoPath = strings.TrimSuffix(urlPath, ".git/info/refs")
		action = "info/refs"
	case strings.HasSuffix(urlPath, ".git/git-upload-pack"):
		repoPath = strings.TrimSuffix(urlPath, ".git/git-upload-pack")
		action = "git-upload-pack"
	case strings.HasSuffix(urlPath, ".git/git-receive-pack"):
		repoPath = strings.TrimSuffix(urlPath, ".git/git-receive-pack")
		action = "git-receive-pack"
	default:
		return "", "", errors.New("not a git path")
	}

	repoPath = strings.Trim(repoPath, "/")
	if repoPath == "" {
		return "", "", errors.New("not a git path")
	}

	parts := strings.SplitSeq(repoPath, "/")
	for part := range parts {
		if part == "" || part == "." || part == ".." || usecase.IsReservedSlug(part) {
			return "", "", errors.New("invalid_repository_path")
		}
	}

	return repoPath, action, nil
}
