package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

type RepositoryHandler struct {
	repos *usecase.RepositoryUseCase
}

func NewRepositoryHandler(repos *usecase.RepositoryUseCase) *RepositoryHandler {
	return &RepositoryHandler{repos: repos}
}

type createRepoRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	FolderID      *uint  `json:"folder_id"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
}

type updateRepoRequest struct {
	Description   *string `json:"description"`
	Private       *bool   `json:"private"`
	DefaultBranch *string `json:"default_branch"`
}

type moveRepoRequest struct {
	FolderID *uint `json:"folder_id"`
}

func (h *RepositoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.repos.Create(usecase.CreateRepositoryInput{
		Name:          req.Name,
		Description:   req.Description,
		OwnerID:       user.ID,
		FolderID:      req.FolderID,
		Private:       req.Private,
		DefaultBranch: req.DefaultBranch,
		BasePath:      "data/repos",
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *RepositoryHandler) List(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	user, loggedIn := middleware.UserFromContext(r.Context())

	if scope == "public" || !loggedIn {
		items, err := h.repos.ListPublic()
		if err != nil {
			writeRepoError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"repos": items})
		return
	}

	items, err := h.repos.ListOwn(user.ID)
	if err != nil {
		writeRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"repos": items})
}

func (h *RepositoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	owner, folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
		writeRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *RepositoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	owner, folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req updateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.repos.Update(usecase.UpdateRepositoryInput{
		OwnerID:       user.ID,
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		Description:   req.Description,
		Private:       req.Private,
		DefaultBranch: req.DefaultBranch,
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *RepositoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	owner, folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.repos.Delete(usecase.ResolveRepositoryInput{
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		ViewerID:      user.ID,
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RepositoryHandler) Move(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	owner, folderPath, name, err := parseRepoMovePath(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req moveRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.repos.Move(usecase.MoveRepositoryInput{
		OwnerID:       user.ID,
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		NewFolderID:   req.FolderID,
		BasePath:      "data/repos",
	})
	if err != nil {
		writeRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func parseRepoPath(r *http.Request) (owner, folderPath, name string, err error) {
	owner = strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		return "", "", "", errors.New("invalid repository path")
	}

	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "", "", "", errors.New("invalid repository path")
	}

	name = parts[len(parts)-1]
	if len(parts) > 1 {
		folderPath = strings.Join(parts[:len(parts)-1], "/")
	}

	return owner, folderPath, name, nil
}

func parseRepoMovePath(r *http.Request) (owner, folderPath, name string, err error) {
	owner = strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		return "", "", "", errors.New("invalid repository path")
	}

	if !strings.HasSuffix(rest, "/move") && rest != "move" {
		return "", "", "", errors.New("use POST /api/repos/{owner}/.../{name}/move")
	}

	rest = strings.TrimSuffix(rest, "/move")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", "", "", errors.New("invalid repository path")
	}

	parts := strings.Split(rest, "/")
	name = parts[len(parts)-1]
	if len(parts) > 1 {
		folderPath = strings.Join(parts[:len(parts)-1], "/")
	}

	return owner, folderPath, name, nil
}

func writeRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrNameRequired),
		errors.Is(err, usecase.ErrInvalidName),
		errors.Is(err, usecase.ErrInvalidBranchName):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrRepoNotFound),
		errors.Is(err, usecase.ErrFolderNotFound),
		errors.Is(err, usecase.ErrRefNotFound),
		errors.Is(err, usecase.ErrPathNotFound),
		errors.Is(err, usecase.ErrEmptyRepo):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrRepoForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
