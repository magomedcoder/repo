package handler

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type RepositoryHandler struct {
	repos  *usecase.RepositoryUseCase
	access *audit.Logger
}

func NewRepositoryHandler(repos *usecase.RepositoryUseCase) *RepositoryHandler {
	return &RepositoryHandler{repos: repos}
}

func (h *RepositoryHandler) SetAccessLogger(log *audit.Logger) {
	h.access = log
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

	if h.access != nil {
		h.access.RepoCreate(c.R, user, out.RepoPath)
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
	offset, limit = usecase.NormalizePagination(offset, limit)
	sortKey := c.Query("sort")
	sqlPage := sortKey == "" || sortKey == "name"
	order := domain.RepoOrderUpdated
	if sortKey == "name" {
		order = domain.RepoOrderName
	}

	fetch := func(lim, off int) ([]usecase.RepositoryItem, error) {
		if q != "" {
			searchPublic := scope == "public" || scope == "home" || !loggedIn
			return h.repos.Search(c.Ctx(), q, ownerID, searchPublic, lim, off, order)
		}

		if scope == "home" || (!loggedIn && scope != "public") {
			return h.repos.ListHome(c.Ctx(), lim, off, order)
		}

		if publicOnly {
			return h.repos.ListPublic(c.Ctx(), lim, off, order)
		}

		return h.repos.ListOwn(c.Ctx(), user.ID, lim, off, order)
	}

	var items []usecase.RepositoryItem
	var err error
	if sqlPage {
		items, err = fetch(limit, offset)
	} else {
		items, err = listReposForSort(fetch)
	}

	if err != nil {
		return mapError(c, err)
	}

	if sortKey != "name" {
		sortRepoItems(items, sortKey)
	}

	if sqlPage {
		return httpx.OK(c, reposResponse{Repos: items})
	}

	return httpx.OK(c, reposResponse{Repos: paginateRepoItems(items, offset, limit)})
}

func listReposForSort(fetch func(limit, offset int) ([]usecase.RepositoryItem, error)) ([]usecase.RepositoryItem, error) {
	const pageSize = 100
	const maxTotal = 5000
	all := make([]usecase.RepositoryItem, 0, pageSize)
	for off := 0; off < maxTotal; off += pageSize {
		page, err := fetch(pageSize, off)
		if err != nil {
			return nil, err
		}

		all = append(all, page...)
		if len(page) < pageSize {
			break
		}
	}

	return all, nil
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

	if h.access != nil {
		h.access.RepoDelete(c.R, user, audit.RepoKey(folderPath, name))
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

	if h.access != nil {
		h.access.RepoMove(c.R, user, audit.RepoKey(folderPath, name), out.RepoPath)
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
