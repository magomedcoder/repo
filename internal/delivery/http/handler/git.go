package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
)

type GitHandler struct {
	git *usecase.GitUseCase
}

func NewGitHandler(git *usecase.GitUseCase) *GitHandler {
	return &GitHandler{git: git}
}

func IsGitHTTPPath(path string) bool {
	return strings.Contains(path, ".git/info/refs") || strings.Contains(path, ".git/git-upload-pack") || strings.Contains(path, ".git/git-receive-pack")
}

func (h *GitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	owner, folderPath, name, action, err := parseGitHTTPPath(r.URL.Path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "info/refs":
		h.infoRefs(w, r, owner, folderPath, name)
	case r.Method == http.MethodPost && action == "git-upload-pack":
		h.servePack(w, r, owner, folderPath, name, usecase.GitUploadPack)
	case r.Method == http.MethodPost && action == "git-receive-pack":
		h.servePack(w, r, owner, folderPath, name, usecase.GitReceivePack)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *GitHandler) infoRefs(w http.ResponseWriter, r *http.Request, owner, folderPath, name string) {
	service, err := usecase.ParseGitService(r.URL.Query().Get("service"))
	if err != nil {
		http.Error(w, "unsupported service", http.StatusForbidden)
		return
	}

	repo, _, err := h.resolveAndAuth(w, r, owner, folderPath, name, service)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/x-"+string(service)+"-advertisement")
	w.Header().Set("Cache-Control", "no-cache")
	if err := h.git.AdvertiseRefs(repo.Path, service, w); err != nil {
		http.Error(w, "git error", http.StatusInternalServerError)
	}
}

func (h *GitHandler) servePack(w http.ResponseWriter, r *http.Request, owner, folderPath, name string, service usecase.GitService) {
	repo, _, err := h.resolveAndAuth(w, r, owner, folderPath, name, service)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "application/x-"+string(service)+"-result")
	w.Header().Set("Cache-Control", "no-cache")
	if err := h.git.ServePack(repo.Path, service, r.Body, w); err != nil {
		http.Error(w, "git error", http.StatusInternalServerError)
		return
	}

	if service == usecase.GitReceivePack {
		_ = h.git.TouchActivity(repo.ID)
	}
}

func (h *GitHandler) resolveAndAuth(
	w http.ResponseWriter,
	r *http.Request,
	owner, folderPath, name string,
	service usecase.GitService,
) (*domain.Repository, *domain.User, error) {
	repo, _, err := h.git.Resolve(owner, folderPath, name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, nil, err
	}

	viewer, authErr := h.authenticateRequest(r)
	needAuth := repo.IsPrivate || service == usecase.GitReceivePack
	if needAuth && authErr != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil, authErr
	}

	if authErr != nil {
		viewer = nil
	}

	if err := h.git.Authorize(repo, viewer, service); err != nil {
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
		return h.git.Authenticate(user, pass)
	}

	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return h.git.Authenticate("", strings.TrimSpace(auth[7:]))
	}

	return nil, usecase.ErrUnauthorized
}

func parseGitHTTPPath(urlPath string) (owner, folderPath, name, action string, err error) {
	urlPath = strings.Trim(urlPath, "/")
	var suffix string
	switch {
	case strings.Contains(urlPath, ".git/info/refs"):
		suffix = ".git/info/refs"
		action = "info/refs"
	case strings.Contains(urlPath, ".git/git-upload-pack"):
		suffix = ".git/git-upload-pack"
		action = "git-upload-pack"
	case strings.Contains(urlPath, ".git/git-receive-pack"):
		suffix = ".git/git-receive-pack"
		action = "git-receive-pack"
	default:
		return "", "", "", "", errors.New("not a git path")
	}

	idx := strings.Index(urlPath, suffix)
	if idx < 0 {
		return "", "", "", "", errors.New("not a git path")
	}

	repoPath := urlPath[:idx] // owner[/folder...]/name
	parts := strings.Split(repoPath, "/")
	if len(parts) < 2 {
		return "", "", "", "", errors.New("invalid repository path")
	}

	owner = parts[0]
	name = parts[len(parts)-1]
	if len(parts) > 2 {
		folderPath = strings.Join(parts[1:len(parts)-1], "/")
	}

	if owner == "" || name == "" || owner == "api" {
		return "", "", "", "", errors.New("invalid repository path")
	}

	return owner, folderPath, name, action, nil
}
