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
	Name            string `json:"name"`
	Description     string `json:"description"`
	Private         bool   `json:"private"`
	ShowOnHome      bool   `json:"show_on_home"`
	DefaultBranch   string `json:"default_branch"`
	MirrorDirection string `json:"mirror_direction"`
	MirrorURL       string `json:"mirror_url"`
	MirrorUsername  string `json:"mirror_username"`
	MirrorPassword  string `json:"mirror_password"`
}

type configureMirrorRequest struct {
	Enabled         *bool   `json:"enabled"`
	Direction       string  `json:"direction"`
	URL             string  `json:"url"`
	Username        *string `json:"username"`
	Password        *string `json:"password"`
	Clear           bool    `json:"clear"`
	ClearPassword   bool    `json:"clear_password"`
	Interval        *string `json:"interval"`
	SyncWindow      *string `json:"sync_window"`
	Force           *bool   `json:"force"`
	Prune           *bool   `json:"prune"`
	Refspecs        *string `json:"refspecs"`
	SSHPrivateKey   *string `json:"ssh_private_key"`
	SSHPassphrase   *string `json:"ssh_passphrase"`
	ClearSSHKey     bool    `json:"clear_ssh_key"`
	InsecureTLS     *bool   `json:"insecure_tls"`
	PinnedHosts     *string `json:"pinned_hosts"`
	MaxBytes        *int64  `json:"max_bytes"`
	ConflictPolicy  *string `json:"conflict_policy"`
	OwnerOnly       *bool   `json:"owner_only"`
	GroupID         *uint   `json:"group_id"`
	VerifySigned    *bool   `json:"verify_signed"`
	CurrentPassword string  `json:"current_password"`
}

type updateRepoRequest struct {
	Name                 *string `json:"name"`
	Description          *string `json:"description"`
	Private              *bool   `json:"private"`
	ShowOnHome           *bool   `json:"show_on_home"`
	DefaultBranch        *string `json:"default_branch"`
	ProtectDefaultBranch *bool   `json:"protect_default_branch"`
	IssuesEnabled        *bool   `json:"issues_enabled"`
	CurrentPassword      string  `json:"current_password"`
}

type deleteRepoRequest struct {
	Confirm         string `json:"confirm"`
	CurrentPassword string `json:"current_password"`
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
		Name:            req.Name,
		Description:     req.Description,
		ViewerID:        user.ID,
		OwnerID:         user.ID,
		Private:         req.Private,
		ShowOnHome:      req.ShowOnHome,
		DefaultBranch:   req.DefaultBranch,
		MirrorDirection: req.MirrorDirection,
		MirrorURL:       req.MirrorURL,
		MirrorUsername:  req.MirrorUsername,
		MirrorPassword:  req.MirrorPassword,
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
	if !loggedIn && h.repos.AnonymousReadDenied() {
		return mapError(c, usecase.ErrUnauthorized)
	}

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

		if scope == "shared" {
			if !loggedIn {
				return nil, usecase.ErrUnauthorized
			}

			return h.repos.ListShared(c.Ctx(), user.ID, lim, off, order)
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
	const maxTotal = 500

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
	name, err := parseRepoName(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	var viewerID uint
	if user, ok := middleware.UserFromContext(c.Ctx()); ok {
		viewerID = user.ID
	}

	out, err := h.repos.Get(c.Ctx(), usecase.ResolveRepositoryInput{
		Name:     name,
		ViewerID: viewerID,
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

	name, err := parseRepoName(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	req, err := httpx.BindJSON[updateRepoRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.repos.Update(c.Ctx(), usecase.UpdateRepositoryInput{
		ViewerID:             user.ID,
		Name:                 name,
		NewName:              req.Name,
		Description:          req.Description,
		Private:              req.Private,
		ShowOnHome:           req.ShowOnHome,
		DefaultBranch:        req.DefaultBranch,
		ProtectDefaultBranch: req.ProtectDefaultBranch,
		IssuesEnabled:        req.IssuesEnabled,
		CurrentPassword:      req.CurrentPassword,
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

	name, err := parseRepoName(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	req, bindErr := httpx.BindJSON[deleteRepoRequest](c)
	if bindErr != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	err = h.repos.Delete(c.Ctx(), usecase.DeleteRepositoryInput{
		Name:            name,
		ViewerID:        user.ID,
		Confirm:         req.Confirm,
		CurrentPassword: req.CurrentPassword,
	})
	if err != nil {
		return mapError(c, err)
	}

	if h.access != nil {
		h.access.RepoDelete(c.R, user, name)
	}

	return c.NoContent()
}

func (h *RepositoryHandler) ConfigureMirror(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	name, err := parseRepoName(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	req, err := httpx.BindJSON[configureMirrorRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	out, err := h.repos.ConfigureMirror(c.Ctx(), usecase.ConfigureMirrorInput{
		ViewerID:        user.ID,
		Name:            name,
		Direction:       req.Direction,
		URL:             req.URL,
		Username:        req.Username,
		Password:        req.Password,
		ClearPassword:   req.ClearPassword,
		Enabled:         enabled,
		Clear:           req.Clear,
		Interval:        req.Interval,
		SyncWindow:      req.SyncWindow,
		Force:           req.Force,
		Prune:           req.Prune,
		Refspecs:        req.Refspecs,
		SSHPrivateKey:   req.SSHPrivateKey,
		SSHPassphrase:   req.SSHPassphrase,
		ClearSSHKey:     req.ClearSSHKey,
		InsecureTLS:     req.InsecureTLS,
		PinnedHosts:     req.PinnedHosts,
		MaxBytes:        req.MaxBytes,
		ConflictPolicy:  req.ConflictPolicy,
		OwnerOnly:       req.OwnerOnly,
		GroupID:         req.GroupID,
		VerifySigned:    req.VerifySigned,
		CurrentPassword: req.CurrentPassword,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) SyncMirror(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	name, err := parseRepoName(c)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	out, err := h.repos.SyncMirror(c.Ctx(), usecase.ResolveRepositoryInput{
		Name:     name,
		ViewerID: user.ID,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func parseRepoName(c *httpx.Context) (string, error) {
	rest := strings.Trim(c.Param("path"), "/")
	if rest != "" {
		return repoNameFromParts(strings.Split(rest, "/"))
	}

	parts := namedRepoParts(c)
	if len(parts) == 0 {
		return "", errors.New("invalid_repository_path")
	}

	return repoNameFromParts(parts)
}
