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
	Organization  string `json:"organization"`
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
		Description:   req.Description,
		ViewerID:      user.ID,
		OwnerID:       user.ID,
		Organization:  req.Organization,
		FolderID:      req.FolderID,
		Private:       req.Private,
		DefaultBranch: req.DefaultBranch,
		BasePath:      "data/repos",
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
		
		writeJSON(w, http.StatusOK, map[string]any{"repos": items})
		return
	}

	if publicOnly {
		items, err := h.repos.ListPublic()
		if err != nil {
			writeRepoError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repos": items})
		return
	}

	items, err := h.repos.ListOwn(user.ID)
	if err != nil {
		writeRepoError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repos": items})
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
		ViewerID:      user.ID,
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		Description:   req.Description,
		Private:       req.Private,
		DefaultBranch: req.DefaultBranch,
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

func (h *RepositoryHandler) Move(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	owner, folderPath, name, err := parseRepoMovePath(r)
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
		BasePath:      "data/repos",
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

func parseRepoMovePath(r *http.Request) (owner, folderPath, name string, err error) {
	owner = strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		return "", "", "", errors.New("invalid_repository_path")
	}

	if !strings.HasSuffix(rest, "/move") && rest != "move" {
		return "", "", "", errors.New("invalid_repo_move_path")
	}

	rest = strings.TrimSuffix(rest, "/move")
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
	case errors.Is(err, usecase.ErrRepoForbidden),
		errors.Is(err, usecase.ErrOrgForbidden):
		writeError(w, r, http.StatusForbidden, err.Error())
	case errors.Is(err, usecase.ErrOrgNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
