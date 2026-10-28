package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"repo.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"repo.magomedcoder.ru/repo/internal/usecase"
)

type RepositoryHandler struct {
	repos     *usecase.RepositoryUseCase
	reposRoot string
}

func NewRepositoryHandler(repos *usecase.RepositoryUseCase, reposRoot string) *RepositoryHandler {
	if reposRoot == "" {
		reposRoot = usecase.DefaultReposRoot
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
	FolderPath    string `json:"folder_path"`
	Private       bool   `json:"private"`
	ShowOnHome    bool   `json:"show_on_home"`
	DefaultBranch string `json:"default_branch"`
}

type updateRepoRequest struct {
	Name                 *string `json:"name"`
	Description          *string `json:"description"`
	Private              *bool   `json:"private"`
	ShowOnHome           *bool   `json:"show_on_home"`
	DefaultBranch        *string `json:"default_branch"`
	ProtectDefaultBranch *bool   `json:"protect_default_branch"`
}

type moveRepoRequest struct {
	FolderPath string `json:"folder_path"`
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
		FolderPath:    req.FolderPath,
		Private:       req.Private,
		ShowOnHome:    req.ShowOnHome,
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
	publicOnly := scope == "public" || scope == "home" || !loggedIn

	var ownerID uint
	if loggedIn {
		ownerID = user.ID
	}

	offset, limit := usecase.ParseOffsetLimit(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
	sortKey := r.URL.Query().Get("sort")

	sqlPage := sortKey == ""
	fetchOff, fetchLim := 0, 0
	if sqlPage {
		fetchOff, fetchLim = offset, limit
	}

	if q != "" {
		searchPublic := scope == "public" || scope == "home" || !loggedIn
		items, err := h.repos.Search(q, ownerID, searchPublic, fetchLim, fetchOff)
		if err != nil {
			writeRepoError(w, r, err)
			return
		}

		sortRepoItems(items, sortKey)
		if sqlPage {
			writeJSON(w, http.StatusOK, map[string]any{"repos": items})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repos": paginateRepoItems(items, offset, limit)})
		return
	}

	if scope == "home" || (!loggedIn && scope != "public") {
		items, err := h.repos.ListHome(fetchLim, fetchOff)
		if err != nil {
			writeRepoError(w, r, err)
			return
		}

		sortRepoItems(items, sortKey)
		if sqlPage {
			writeJSON(w, http.StatusOK, map[string]any{"repos": items})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repos": paginateRepoItems(items, offset, limit)})
		return
	}

	if publicOnly {
		items, err := h.repos.ListPublic(fetchLim, fetchOff)
		if err != nil {
			writeRepoError(w, r, err)
			return
		}
		sortRepoItems(items, sortKey)
		if sqlPage {
			writeJSON(w, http.StatusOK, map[string]any{"repos": items})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"repos": paginateRepoItems(items, offset, limit)})
		return
	}

	items, err := h.repos.ListOwn(user.ID, fetchLim, fetchOff)
	if err != nil {
		writeRepoError(w, r, err)
		return
	}
	sortRepoItems(items, sortKey)
	if sqlPage {
		writeJSON(w, http.StatusOK, map[string]any{"repos": items})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repos": paginateRepoItems(items, offset, limit)})
}

func paginateRepoItems(items []usecase.RepositoryItem, offset, limit int) []usecase.RepositoryItem {
	if limit <= 0 && offset <= 0 {
		return items
	}

	if offset < 0 {
		offset = 0
	}

	if offset >= len(items) {
		return nil
	}

	end := len(items)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}

	return items[offset:end]
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
	folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var viewerID uint
	if user, ok := middleware.UserFromContext(r.Context()); ok {
		viewerID = user.ID
	}

	out, err := h.repos.Get(usecase.ResolveRepositoryInput{
		FolderPath: folderPath,
		Name:       name,
		ViewerID:   viewerID,
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

	folderPath, name, err := parseRepoPath(r)
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
		FolderPath:           folderPath,
		Name:                 name,
		NewName:              req.Name,
		Description:          req.Description,
		Private:              req.Private,
		ShowOnHome:           req.ShowOnHome,
		DefaultBranch:        req.DefaultBranch,
		ProtectDefaultBranch: req.ProtectDefaultBranch,
		BasePath:             h.reposRoot,
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

	folderPath, name, err := parseRepoPath(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	err = h.repos.Delete(usecase.ResolveRepositoryInput{
		FolderPath: folderPath,
		Name:       name,
		ViewerID:   user.ID,
	})
	if err != nil {
		writeRepoError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RepositoryHandler) MoveWithParts(w http.ResponseWriter, r *http.Request, repoParts []string) {
	folderPath, name, err := folderAndName(repoParts)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	h.move(w, r, folderPath, name)
}

func (h *RepositoryHandler) move(w http.ResponseWriter, r *http.Request, folderPath, name string) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req moveRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.repos.Move(usecase.MoveRepositoryInput{
		ViewerID:      user.ID,
		FolderPath:    folderPath,
		Name:          name,
		NewFolderPath: req.FolderPath,
		BasePath:      h.reposRoot,
	})
	if err != nil {
		writeRepoError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func parseRepoPath(r *http.Request) (folderPath, name string, err error) {
	rest := strings.Trim(r.PathValue("path"), "/")
	if rest == "" {
		return "", "", errors.New("invalid_repository_path")
	}
	return folderAndName(strings.Split(rest, "/"))
}

func writeRepoError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrNameRequired),
		errors.Is(err, usecase.ErrInvalidName),
		errors.Is(err, usecase.ErrInvalidBranchName),
		errors.Is(err, usecase.ErrReservedSlug):
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
