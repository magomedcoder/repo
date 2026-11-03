package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
)

const uploadPackMaxBytes = 32 << 20

type GitHandler struct {
	git             *usecase.GitUseCase
	access          *audit.Logger
	receiveMaxBytes int64
}

func NewGitHandler(git *usecase.GitUseCase) *GitHandler {
	return &GitHandler{git: git, receiveMaxBytes: 512 << 20}
}

func (h *GitHandler) SetAccessLogger(log *audit.Logger) {
	h.access = log
}

func (h *GitHandler) SetTempDir(string) {}

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

	var (
		body    io.Reader = http.MaxBytesReader(w, r.Body, uploadPackMaxBytes)
		receive *domain.ReceivePackOptions
	)

	if service == usecase.GitReceivePack {
		unlock := h.git.LockRepo(repo.Path)
		defer unlock()

		limit := h.receiveMaxBytes
		if limit <= 0 {
			limit = 512 << 20
		}

		body = http.MaxBytesReader(w, r.Body, limit)
		if repo.ProtectDefaultBranch {
			receive = &domain.ReceivePackOptions{
				ProtectDefaultBranch: true,
				DefaultBranch:        repo.DefaultBranch,
			}
		}
	}

	w.Header().Set("Content-Type", "application/x-"+string(service)+"-result")
	w.Header().Set("Cache-Control", "no-cache")
	if err := h.git.ServePack(r.Context(), repo.Path, service, body, w, receive); err != nil {
		if errors.Is(err, domain.ErrBranchProtected) {
			return
		}
		http.Error(w, "git error", http.StatusInternalServerError)
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
	}
}

func (h *GitHandler) resolveAndAuth(w http.ResponseWriter, r *http.Request, repoPath string, service usecase.GitService) (*domain.Repository, *domain.User, error) {
	repo, err := h.git.ResolveClonePath(r.Context(), repoPath)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, nil, err
	}

	viewer, authErr := h.authenticateRequest(r)
	needAuth := repo.IsPrivate || service == usecase.GitReceivePack
	if needAuth && authErr != nil {
		if h.access != nil {
			login, _, _ := r.BasicAuth()
			h.access.FailedAuth(r, login)
		}

		if errors.Is(authErr, usecase.ErrTooManyAttempts) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			return nil, nil, authErr
		}

		if repo.IsPrivate {
			http.Error(w, "not found", http.StatusNotFound)
			return nil, nil, authErr
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, authErr
	}

	if authErr != nil {
		viewer = nil
	}

	if err := h.git.Authorize(r.Context(), repo, viewer, service); err != nil {
		if repo.IsPrivate {
			http.Error(w, "not found", http.StatusNotFound)
			return nil, nil, err
		}

		if errors.Is(err, usecase.ErrUnauthorized) {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil, nil, err
		}

		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, nil, err
	}

	return repo, viewer, nil
}

func (h *GitHandler) authenticateRequest(r *http.Request) (*domain.User, error) {
	if user, pass, ok := r.BasicAuth(); ok {
		u, err := h.git.Authenticate(r.Context(), middleware.ClientIPKey(r), user, pass)
		if errors.Is(err, usecase.ErrTooManyAttempts) {
			return nil, err
		}
		return u, err
	}

	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return h.git.Authenticate(r.Context(), middleware.ClientIPKey(r), "", strings.TrimSpace(auth[7:]))
	}

	return nil, usecase.ErrUnauthorized
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
