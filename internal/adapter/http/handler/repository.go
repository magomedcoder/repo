package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"git.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"git.magomedcoder.ru/repo/internal/usecase"
)

type RepositoryHandler struct {
	repos     *usecase.RepositoryUseCase
	reposRoot string
}

func NewRepositoryHandler(repos *usecase.RepositoryUseCase, reposRoot string) *RepositoryHandler {
	if reposRoot == "" {
		reposRoot = "data/repos"
	}

	return &RepositoryHandler{
		repos:     repos,
		reposRoot: reposRoot,
	}
}

type createRepoRequest struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Description   string `json:"description"`
	FolderID      *uint  `json:"folder_id"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
}

type updateRepoRequest struct {
	Description          *string `json:"description"`
	Private              *bool   `json:"private"`
	DefaultBranch        *string `json:"default_branch"`
	ProtectDefaultBranch *bool   `json:"protect_default_branch"`
}

type moveRepoRequest struct {
	FolderID *uint `json:"folder_id"`
}

func (h *RepositoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.repos.Create(usecase.CreateRepositoryInput{
		Name:          req.Name,
		Path:          req.Path,
		Description:   req.Description,
		ViewerID:      user.ID,
		OwnerID:       user.ID,
		FolderID:      req.FolderID,
		Private:       req.Private,
		DefaultBranch: req.DefaultBranch,
		BasePath:      h.reposRoot,
	})
	if err != nil {
		writeRepoError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *RepositoryHandler) List(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	user, loggedIn := middleware.UserFromContext(r.Context())
	publicOnly := scope == "public" || !loggedIn

	var ownerID uint
	if loggedIn {
		ownerID = user.ID
	}

	if q != "" {
		items, err := h.repos.Search(q, ownerID, publicOnly)
		if err != nil {
			writeRepoError(w, r, err)
			return
		}

		sortRepoItems(items, r.URL.Query().Get("sort"))
		writeJSON(w, http.StatusOK, map[string]any{"repos": items})
		return
	}

	if publicOnly {
		items, err := h.repos.ListPublic()
		if err != nil {
			writeRepoError(w, r, err)
			return
		}
		sortRepoItems(items, r.URL.Query().Get("sort"))
		writeJSON(w, http.StatusOK, map[string]any{"repos": items})
		return
	}

	items, err := h.repos.ListOwn(user.ID)
	if err != nil {
		writeRepoError(w, r, err)
		return
	}
	sortRepoItems(items, r.URL.Query().Get("sort"))
	writeJSON(w, http.StatusOK, map[string]any{"repos": items})
}

func sortRepoItems(items []usecase.RepositoryItem, key string) {
	switch key {
	case "description":
		slices.SortFunc(items, func(a, b usecase.RepositoryItem) int {
			return strings.Compare(strings.ToLower(a.Description), strings.ToLower(b.Description))
		})
	case "idle":
		slices.SortFunc(items, func(a, b usecase.RepositoryItem) int {
			var at, bt int64
			if a.LastActivityAt != nil {
				at = a.LastActivityAt.Unix()
			}
			if b.LastActivityAt != nil {
				bt = b.LastActivityAt.Unix()
			}
			if at == bt {
				return 0
			}
			if at < bt {
				return -1
			}
			return 1
		})
	default:
		slices.SortFunc(items, func(a, b usecase.RepositoryItem) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})
	}
}

func (h *RepositoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	owner, folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var viewerID uint
	if user, ok := middleware.UserFromContext(r.Context()); ok {
		viewerID = user.ID
	}

	out, err := h.repos.Get(usecase.ResolveRepositoryInput{
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		ViewerID:      viewerID,
	})
	if err != nil {
		if viewerID == 0 && (errors.Is(err, usecase.ErrRepoNotFound) || errors.Is(err, usecase.ErrUnauthorized) || errors.Is(err, usecase.ErrRepoForbidden)) {
			writeError(w, r, http.StatusUnauthorized, "unauthorized")
			return
		}
		writeRepoError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *RepositoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	owner, folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var req updateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.repos.Update(usecase.UpdateRepositoryInput{
		ViewerID:             user.ID,
		OwnerUsername:        owner,
		FolderPath:           folderPath,
		Name:                 name,
		Description:          req.Description,
		Private:              req.Private,
		DefaultBranch:        req.DefaultBranch,
		ProtectDefaultBranch: req.ProtectDefaultBranch,
	})
	if err != nil {
		writeRepoError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *RepositoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	owner, folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = h.repos.Delete(usecase.ResolveRepositoryInput{
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		ViewerID:      user.ID,
	})
	if err != nil {
		writeRepoError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RepositoryHandler) DispatchPost(w http.ResponseWriter, r *http.Request) {
	h.Move(w, r)
}

func (h *RepositoryHandler) Move(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	owner, folderPath, name, err := parseRepoActionPath(r, "move")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var req moveRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.repos.Move(usecase.MoveRepositoryInput{
		ViewerID:      user.ID,
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		NewFolderID:   req.FolderID,
		BasePath:      h.reposRoot,
	})
	if err != nil {
		writeRepoError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func parseRepoPath(r *http.Request) (owner, folderPath, name string, err error) {
	owner = strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		return "", "", "", errors.New("invalid_repository_path")
	}

	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "", "", "", errors.New("invalid_repository_path")
	}

	name = parts[len(parts)-1]
	if len(parts) > 1 {
		folderPath = strings.Join(parts[:len(parts)-1], "/")
	}

	return owner, folderPath, name, nil
}

func parseRepoActionPath(r *http.Request, action string) (owner, folderPath, name string, err error) {
	owner = strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		return "", "", "", errors.New("invalid_repository_path")
	}

	suffix := "/" + action
	if !strings.HasSuffix(rest, suffix) && rest != action {
		return "", "", "", errors.New("invalid_repo_move_path")
	}

	rest = strings.TrimSuffix(rest, suffix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", "", "", errors.New("invalid_repository_path")
	}

	parts := strings.Split(rest, "/")
	name = parts[len(parts)-1]
	if len(parts) > 1 {
		folderPath = strings.Join(parts[:len(parts)-1], "/")
	}

	return owner, folderPath, name, nil
}

func writeRepoError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrNameRequired),
		errors.Is(err, usecase.ErrInvalidName),
		errors.Is(err, usecase.ErrInvalidBranchName):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrAlreadyExists):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrRepoNotFound),
		errors.Is(err, usecase.ErrFolderNotFound),
		errors.Is(err, usecase.ErrRefNotFound),
		errors.Is(err, usecase.ErrPathNotFound),
		errors.Is(err, usecase.ErrEmptyRepo):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrRepoForbidden):
		writeError(w, r, http.StatusForbidden, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
