package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type checkMirrorRequest struct {
	URL      string  `json:"url"`
	Username *string `json:"username"`
	Password *string `json:"password"`
	DryRun   bool    `json:"dry_run"`
}

type mirrorTargetRequest struct {
	URL      string  `json:"url"`
	Username *string `json:"username"`
	Password *string `json:"password"`
	Enabled  *bool   `json:"enabled"`
}

type bulkMirrorRequest struct {
	RepoIDs []uint `json:"repo_ids"`
	IDs     []uint `json:"ids"`
	Pause   *bool  `json:"pause"`
	Paused  *bool  `json:"paused"`
}

type mirrorGroupRequest struct {
	Name        string `json:"name"`
	UpstreamURL string `json:"upstream_url"`
	Direction   string `json:"direction"`
	Interval    string `json:"interval"`
}

type importMirrorsRequest struct {
	FolderPath string   `json:"folder_path"`
	URLs       []string `json:"urls"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Private    bool     `json:"private"`
	GroupID    uint     `json:"group_id"`
	Interval   string   `json:"interval"`
}

func (h *RepositoryHandler) mirrorInput(c *httpx.Context) (usecase.ResolveRepositoryInput, error) {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return usecase.ResolveRepositoryInput{}, usecase.ErrUnauthorized
	}

	folderPath, name, err := parseRepoPath(c)
	if err != nil {
		return usecase.ResolveRepositoryInput{}, usecase.ErrRepoNotFound
	}

	return usecase.ResolveRepositoryInput{
		FolderPath: folderPath,
		Name:       name,
		ViewerID:   user.ID,
	}, nil
}

func (h *RepositoryHandler) MirrorStatus(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	out, err := h.repos.GetMirrorStatus(c.Ctx(), in)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) MirrorHistory(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	limit, _ := strconv.Atoi(c.Query("limit"))

	out, err := h.repos.MirrorHistory(c.Ctx(), in, limit)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"events": out})
}

func (h *RepositoryHandler) CheckMirror(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	req, _ := httpx.BindJSON[checkMirrorRequest](c)

	out, err := h.repos.CheckMirror(c.Ctx(), usecase.CheckMirrorInput{
		ResolveRepositoryInput: in,
		URL:                    req.URL,
		Username:               req.Username,
		Password:               req.Password,
		DryRun:                 req.DryRun,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) VerifyMirror(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	out, err := h.repos.VerifyMirror(c.Ctx(), usecase.VerifyMirrorInput{
		ResolveRepositoryInput: in,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) DryRunMirror(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	out, err := h.repos.CheckMirror(c.Ctx(), usecase.CheckMirrorInput{
		ResolveRepositoryInput: in,
		DryRun:                 true,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{
		"refs":           out.WouldUpdate,
		"head":           out.Head,
		"default_branch": out.DefaultBranch,
		"ok":             out.OK,
		"error":          out.Error,
	})
}

func (h *RepositoryHandler) ListMirrorTargets(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	out, err := h.repos.ListMirrorTargets(c.Ctx(), in)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"targets": out})
}

func (h *RepositoryHandler) SaveMirrorTarget(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	req, err := httpx.BindJSON[mirrorTargetRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	var targetID uint
	if raw := c.Param("target"); raw != "" {
		id, perr := strconv.ParseUint(raw, 10, 32)
		if perr != nil {
			return c.Error(http.StatusBadRequest, "invalid_id")
		}

		targetID = uint(id)
	}

	out, err := h.repos.SaveMirrorTarget(c.Ctx(), usecase.MirrorTargetInput{
		ResolveRepositoryInput: in,
		TargetID:               targetID,
		URL:                    req.URL,
		Username:               req.Username,
		Password:               req.Password,
		Enabled:                req.Enabled,
	})
	if err != nil {
		return mapError(c, err)
	}

	if targetID == 0 {
		return httpx.Created(c, out)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) DeleteMirrorTarget(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	id, perr := strconv.ParseUint(c.Param("target"), 10, 32)
	if perr != nil {
		return c.Error(http.StatusBadRequest, "invalid_id")
	}

	if err := h.repos.DeleteMirrorTarget(c.Ctx(), in, uint(id)); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *RepositoryHandler) AdminListMirrors(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	offset, limit := usecase.ParseOffsetLimit(c.Query("offset"), c.Query("limit"))
	offset, limit = usecase.NormalizePagination(offset, limit)

	errFlag := c.Query("errors")
	if errFlag == "" {
		errFlag = c.Query("error")
	}

	filter := domain.MirrorListFilter{
		Direction: c.Query("direction"),
		Status:    c.Query("status"),
		OnlyError: errFlag == "1" || errFlag == "true",
		Query:     strings.TrimSpace(c.Query("q")),
		Limit:     limit,
		Offset:    offset,
	}

	if raw := c.Query("group_id"); raw != "" {
		if id, err := strconv.ParseUint(raw, 10, 32); err == nil {
			filter.GroupID = uint(id)
		}
	}

	out, err := h.repos.ListAdminMirrors(c.Ctx(), user.ID, filter)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"mirrors": out})
}

func (h *RepositoryHandler) AdminVerifyMirrors(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	filter := domain.MirrorListFilter{
		Direction: c.Query("direction"),
		Status:    c.Query("status"),
		Query:     strings.TrimSpace(c.Query("q")),
		Limit:     100,
	}

	out, err := h.repos.AdminVerifyMirrors(c.Ctx(), user.ID, filter)
	if err != nil {
		return mapError(c, err)
	}

	failed := 0
	for _, item := range out {
		if !item.OK || item.Error != "" {
			failed++
		}
	}

	return httpx.OK(c, map[string]any{"results": out, "checked": len(out), "failed": failed})
}

func (h *RepositoryHandler) AdminBulkMirrors(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[bulkMirrorRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	ids := req.RepoIDs
	if len(ids) == 0 {
		ids = req.IDs
	}

	var n int

	switch c.Param("action") {
	case "pause":
		pause := true
		if req.Pause != nil {
			pause = *req.Pause
		} else if req.Paused != nil {
			pause = *req.Paused
		}

		if len(ids) == 0 {
			return c.Error(http.StatusBadRequest, "invalid_json")
		}

		n, err = h.repos.BulkPauseMirrors(c.Ctx(), user.ID, ids, pause)
	case "sync":
		if len(ids) == 0 {
			n, err = h.repos.BulkSyncAllMirrors(c.Ctx(), user.ID)
		} else {
			n, err = h.repos.BulkSyncMirrors(c.Ctx(), user.ID, ids)
		}
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}

	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]int{"affected": n, "queued": n})
}

func (h *RepositoryHandler) ExportMirrorBundle(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	c.W.Header().Set("Content-Type", "application/octet-stream")
	c.W.Header().Set("Content-Disposition", `attachment; filename="mirror.bundle"`)

	if err := h.repos.ExportMirrorBundle(c.Ctx(), in, c.W); err != nil {
		return mapError(c, err)
	}

	return nil
}

func (h *RepositoryHandler) ImportMirrorBundle(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	if err := h.repos.ImportMirrorBundle(c.Ctx(), in, io.LimitReader(c.R.Body, 2<<30)); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *RepositoryHandler) ListMirrorGroups(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.repos.ListMirrorGroups(c.Ctx(), user.ID)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"groups": out})
}

func (h *RepositoryHandler) CreateMirrorGroup(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[mirrorGroupRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.repos.CreateMirrorGroup(c.Ctx(), user.ID, domain.MirrorGroup{
		Name:        req.Name,
		UpstreamURL: req.UpstreamURL,
		Direction:   req.Direction,
		Interval:    req.Interval,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *RepositoryHandler) DeleteMirrorGroup(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_id")
	}

	if err := h.repos.DeleteMirrorGroup(c.Ctx(), user.ID, uint(id)); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *RepositoryHandler) ImportOrgMirrors(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[importMirrorsRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.repos.ImportOrgMirrors(c.Ctx(), usecase.ImportMirrorsInput{
		ViewerID:   user.ID,
		FolderPath: req.FolderPath,
		URLs:       req.URLs,
		Username:   req.Username,
		Password:   req.Password,
		Private:    req.Private,
		GroupID:    req.GroupID,
		Interval:   req.Interval,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"results": out})
}

func (h *RepositoryHandler) MirrorHealth(c *httpx.Context) error {
	secs, _ := strconv.Atoi(c.Query("older_than"))

	out, err := h.repos.MirrorHealth(c.Ctx(), time.Duration(secs)*time.Second)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *RepositoryHandler) MirrorMetrics(c *httpx.Context) error {
	if m := h.repos.MirrorMetrics(); m != nil {
		m.Handler().ServeHTTP(c.W, c.R)

		return nil
	}

	return c.Error(http.StatusNotFound, "not_found")
}
