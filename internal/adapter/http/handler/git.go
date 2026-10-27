package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"git.magomedcoder.ru/repo/internal/domain"
	"git.magomedcoder.ru/repo/internal/usecase"
)

type GitHandler struct {
	git             *usecase.GitUseCase
	receiveMaxBytes int64
}

func NewGitHandler(git *usecase.GitUseCase) *GitHandler {
	return &GitHandler{git: git, receiveMaxBytes: 512 << 20}
}

func (h *GitHandler) SetReceiveMaxBytes(n int64) {
	if n > 0 {
		h.receiveMaxBytes = n
	}
}

func IsGitHTTPPath(path string) bool {
	return strings.Contains(path, ".git/info/refs") || strings.Contains(path, ".git/git-upload-pack") || strings.Contains(path, ".git/git-receive-pack")
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
	if err := h.git.AdvertiseRefs(repo.Path, service, w); err != nil {
		http.Error(w, "git error", http.StatusInternalServerError)
	}
}

func (h *GitHandler) servePack(w http.ResponseWriter, r *http.Request, repoPath string, service usecase.GitService) {
	repo, viewer, err := h.resolveAndAuth(w, r, repoPath, service)
	if err != nil {
		return
	}

	var body io.Reader = r.Body
	if service == usecase.GitReceivePack {
		limit := h.receiveMaxBytes
		if limit <= 0 {
			limit = 512 << 20
		}

		if repo.ProtectDefaultBranch {
			tmp, err := os.CreateTemp("", "receive-pack-")
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			defer os.Remove(tmp.Name())
			defer tmp.Close()

			limited := io.LimitReader(r.Body, limit+1)
			n, err := io.Copy(tmp, limited)
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if n > limit {
				http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
				return
			}
			if _, err := tmp.Seek(0, io.SeekStart); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			cmds, err := readReceiveCommands(tmp)
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			for _, cmd := range cmds {
				ancestor := true
				if !domain.IsZeroHash(cmd[0]) && !domain.IsZeroHash(cmd[1]) {
					ok, err := h.git.Ancestor(repo.Path, cmd[0], cmd[1])
					if err != nil || !ok {
						ancestor = false
					}
				}
				if err := domain.CheckRefUpdate(true, repo.DefaultBranch, cmd[2], cmd[0], cmd[1], ancestor); err != nil {
					http.Error(w, "branch protected", http.StatusForbidden)
					return
				}
			}
			body = tmp
		} else {
			body = http.MaxBytesReader(w, r.Body, limit)
		}
	}

	w.Header().Set("Content-Type", "application/x-"+string(service)+"-result")
	w.Header().Set("Cache-Control", "no-cache")
	if err := h.git.ServePack(repo.Path, service, body, w); err != nil {
		http.Error(w, "git error", http.StatusInternalServerError)
		return
	}

	if service == usecase.GitReceivePack {
		var actorID uint
		if viewer != nil {
			actorID = viewer.ID
		}
		_ = h.git.TouchActivity(repo, actorID)
		_ = h.git.EnsureDefaultBranch(repo.Path, repo.DefaultBranch)
	}
}

func (h *GitHandler) resolveAndAuth(w http.ResponseWriter, r *http.Request, repoPath string, service usecase.GitService) (*domain.Repository, *domain.User, error) {
	repo, err := h.git.ResolveClonePath(repoPath)
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

func parseGitHTTPPath(urlPath string) (repoPath, action string, err error) {
	if strings.Contains(strings.ToLower(urlPath), "%2e") {
		return "", "", errors.New("invalid_repository_path")
	}

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
		return "", "", errors.New("not a git path")
	}

	idx := strings.Index(urlPath, suffix)
	if idx <= 0 {
		return "", "", errors.New("not a git path")
	}

	repoPath = strings.Trim(urlPath[:idx], "/")
	parts := strings.Split(repoPath, "/")
	if len(parts) == 0 {
		return "", "", errors.New("invalid_repository_path")
	}

	for _, part := range parts {
		if part == "" || part == "." || part == ".." || part == "api" {
			return "", "", errors.New("invalid_repository_path")
		}
	}

	return repoPath, action, nil
}

func readReceiveCommands(r io.Reader) (cmds [][3]string, err error) {
	for {
		var lenBuf [4]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return cmds, err
		}
		n, perr := strconv.ParseInt(string(lenBuf[:]), 16, 32)
		if perr != nil || n < 4 {
			return cmds, errors.New("invalid pkt-line")
		}
		if n == 4 {
			break
		}
		payload := make([]byte, n-4)
		if _, err := io.ReadFull(r, payload); err != nil {
			return cmds, err
		}
		line := strings.TrimRight(string(payload), "\n")
		if i := strings.IndexByte(line, 0); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			cmds = append(cmds, [3]string{fields[0], fields[1], fields[2]})
		}
	}
	return cmds, nil
}
