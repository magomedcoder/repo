package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

type adminBulkMirrorRequest struct {
	RepoIDs []uint `json:"repo_ids"`
	Paused  *bool  `json:"paused"`
}

type adminMirrorGroupRequest struct {
	Name        string `json:"name"`
	UpstreamURL string `json:"upstream_url"`
	Direction   string `json:"direction"`
	Interval    string `json:"interval"`
}

type adminImportMirrorsRequest struct {
	URLs     []string `json:"urls"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Private  bool     `json:"private"`
	GroupID  uint     `json:"group_id"`
	Interval string   `json:"interval"`
}

func mirrorListFilterFromQuery(c *httpx.Context, limit, offset int) domain.MirrorListFilter {
	errFlag := c.Query("errors")
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

	return filter
}

func (h *AdminHandler) requireAdminMirrors(c *httpx.Context) (*domain.User, error) {
	if h.mirrors == nil {
		return nil, c.Error(http.StatusNotFound, "not_found")
	}

	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return nil, c.Error(http.StatusUnauthorized, "unauthorized")
	}

	return user, nil
}

func (h *AdminHandler) ListMirrors(c *httpx.Context) error {
	user, err := h.requireAdminMirrors(c)
	if err != nil {
		return err
	}

	offset, limit := usecase.ParseOffsetLimit(c.Query("offset"), c.Query("limit"))
	offset, limit = usecase.NormalizePagination(offset, limit)

	out, err := h.mirrors.List(c.Ctx(), user.ID, mirrorListFilterFromQuery(c, limit, offset))
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"mirrors": out})
}

func (h *AdminHandler) VerifyMirrors(c *httpx.Context) error {
	user, err := h.requireAdminMirrors(c)
	if err != nil {
		return err
	}

	filter := mirrorListFilterFromQuery(c, 100, 0)
	filter.OnlyError = false

	out, err := h.mirrors.Verify(c.Ctx(), user.ID, filter)
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

func (h *AdminHandler) BulkMirrors(c *httpx.Context) error {
	if h.mirrors == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[adminBulkMirrorRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	ids := req.RepoIDs

	var n int

	switch c.Param("action") {
	case "pause":
		if req.Paused == nil || len(ids) == 0 {
			return c.Error(http.StatusBadRequest, "invalid_json")
		}

		n, err = h.mirrors.BulkPause(c.Ctx(), user.ID, ids, *req.Paused)
	case "sync":
		if len(ids) == 0 {
			n, err = h.mirrors.BulkSync(c.Ctx(), user.ID, nil)
		} else {
			n, err = h.mirrors.BulkSync(c.Ctx(), user.ID, ids)
		}
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}

	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]int{"affected": n, "queued": n})
}

func (h *AdminHandler) ListMirrorGroups(c *httpx.Context) error {
	if h.mirrors == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.mirrors.ListGroups(c.Ctx(), user.ID)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"groups": out})
}

func (h *AdminHandler) CreateMirrorGroup(c *httpx.Context) error {
	if h.mirrors == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[adminMirrorGroupRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.mirrors.CreateGroup(c.Ctx(), user.ID, domain.MirrorGroup{
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

func (h *AdminHandler) DeleteMirrorGroup(c *httpx.Context) error {
	if h.mirrors == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_id")
	}

	if err := h.mirrors.DeleteGroup(c.Ctx(), user.ID, uint(id)); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *AdminHandler) ImportOrgMirrors(c *httpx.Context) error {
	if h.mirrors == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[adminImportMirrorsRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.mirrors.ImportOrg(c.Ctx(), usecase.ImportMirrorsInput{
		ViewerID: user.ID,
		URLs:     req.URLs,
		Username: req.Username,
		Password: req.Password,
		Private:  req.Private,
		GroupID:  req.GroupID,
		Interval: req.Interval,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"results": out})
}

func (h *AdminHandler) MirrorHealth(c *httpx.Context) error {
	if h.mirrors == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	secs, _ := strconv.Atoi(c.Query("older_than"))

	out, err := h.mirrors.Health(c.Ctx(), user.ID, time.Duration(secs)*time.Second)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) MirrorMetrics(c *httpx.Context) error {
	if h.mirrors == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	if m := h.mirrors.Metrics(); m != nil {
		m.Handler().ServeHTTP(c.W, c.R)
		return nil
	}

	return c.Error(http.StatusNotFound, "not_found")
}
