package handler

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type RepositoryHandler struct {
	repos *usecase.RepositoryUseCase
}

func NewRepositoryHandler(repos *usecase.RepositoryUseCase) *RepositoryHandler {
	return &RepositoryHandler{repos: repos}
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
	IssuesEnabled        *bool   `json:"issues_enabled"`
}

type moveRepoRequest struct {
	FolderPath string `json:"folder_path"`
}

func (h *RepositoryHandler) Create(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[createRepoRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.repos.Create(c.Ctx(), usecase.CreateRepositoryInput{
		Name:          req.Name,
		Path:          req.Path,
		Description:   req.Description,
		ViewerID:      user.ID,
		OwnerID:       user.ID,
		FolderPath:    req.FolderPath,
		Private:       req.Private,
		ShowOnHome:    req.ShowOnHome,
		DefaultBranch: req.DefaultBranch,
	})
	if err != nil {
		return mapError(c, err)
	}
	return httpx.Created(c, out)
}

func (h *RepositoryHandler) List(c *httpx.Context) error {
	scope := c.Query("scope")
	q := strings.TrimSpace(c.Query("q"))
	user, loggedIn := middleware.UserFromContext(c.Ctx())
	publicOnly := scope == "public" || scope == "home" || !loggedIn

	var ownerID uint
	if loggedIn {
		ownerID = user.ID
	}

	offset, limit := usecase.ParseOffsetLimit(c.Query("offset"), c.Query("limit"))
	sortKey := c.Query("sort")

	sqlPage := sortKey == ""
	fetchOff, fetchLim := 0, 0
	if sqlPage {
		fetchOff, fetchLim = offset, limit
	}

	if q != "" {
		searchPublic := scope == "public" || scope == "home" || !loggedIn
		items, err := h.repos.Search(c.Ctx(), q, ownerID, searchPublic, fetchLim, fetchOff)
		if err != nil {
			return mapError(c, err)
		}

		sortRepoItems(items, sortKey)
		if sqlPage {
			return httpx.OK(c, reposResponse{Repos: items})
		}
		return httpx.OK(c, reposResponse{Repos: paginateRepoItems(items, offset, limit)})
	}

	if scope == "home" || (!loggedIn && scope != "public") {
		items, err := h.repos.ListHome(c.Ctx(), fetchLim, fetchOff)
		if err != nil {
			return mapError(c, err)
		}

		sortRepoItems(items, sortKey)
		if sqlPage {
			return httpx.OK(c, reposResponse{Repos: items})
		}
		return httpx.OK(c, reposResponse{Repos: paginateRepoItems(items, offset, limit)})
	}

	if publicOnly {
		items, err := h.repos.ListPublic(c.Ctx(), fetchLim, fetchOff)
		if err != nil {
			return mapError(c, err)
		}
		sortRepoItems(items, sortKey)
		if sqlPage {
			return httpx.OK(c, reposResponse{Repos: items})
		}
		return httpx.OK(c, reposResponse{Repos: paginateRepoItems(items, offset, limit)})
	}

	items, err := h.repos.ListOwn(c.Ctx(), user.ID, fetchLim, fetchOff)
	if err != nil {
		return mapError(c, err)
	}
	sortRepoItems(items, sortKey)
	if sqlPage {
		return httpx.OK(c, reposResponse{Repos: items})
	}
	return httpx.OK(c, reposResponse{Repos: paginateRepoItems(items, offset, limit)})
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

func (h *RepositoryHandler) Get(c *httpx.Context) error {
	folderPath, name, err := parseRepoPath(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	var viewerID uint
	if user, ok := middleware.UserFromContext(c.Ctx()); ok {
		viewerID = user.ID
	}

	out, err := h.repos.Get(c.Ctx(), usecase.ResolveRepositoryInput{
		FolderPath: folderPath,
		Name:       name,
		ViewerID:   viewerID,
	})
	if err != nil {
		if viewerID == 0 && (errors.Is(err, usecase.ErrRepoNotFound) || errors.Is(err, usecase.ErrUnauthorized) || errors.Is(err, usecase.ErrRepoForbidden)) {
			return c.Error(http.StatusUnauthorized, "unauthorized")
		}
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) Update(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	folderPath, name, err := parseRepoPath(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	req, err := httpx.BindJSON[updateRepoRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.repos.Update(c.Ctx(), usecase.UpdateRepositoryInput{
		ViewerID:             user.ID,
		FolderPath:           folderPath,
		Name:                 name,
		NewName:              req.Name,
		Description:          req.Description,
		Private:              req.Private,
		ShowOnHome:           req.ShowOnHome,
		DefaultBranch:        req.DefaultBranch,
		ProtectDefaultBranch: req.ProtectDefaultBranch,
		IssuesEnabled:        req.IssuesEnabled,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) Delete(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	folderPath, name, err := parseRepoPath(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	err = h.repos.Delete(c.Ctx(), usecase.ResolveRepositoryInput{
		FolderPath: folderPath,
		Name:       name,
		ViewerID:   user.ID,
	})
	if err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *RepositoryHandler) MoveWithParts(c *httpx.Context, repoParts []string) error {
	folderPath, name, err := folderAndName(repoParts)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}
	return h.move(c, folderPath, name)
}

func (h *RepositoryHandler) move(c *httpx.Context, folderPath, name string) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[moveRepoRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.repos.Move(c.Ctx(), usecase.MoveRepositoryInput{
		ViewerID:      user.ID,
		FolderPath:    folderPath,
		Name:          name,
		NewFolderPath: req.FolderPath,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func parseRepoPath(c *httpx.Context) (folderPath, name string, err error) {
	rest := strings.Trim(c.Param("path"), "/")
	if rest == "" {
		return "", "", errors.New("invalid_repository_path")
	}

	return folderAndName(strings.Split(rest, "/"))
}
