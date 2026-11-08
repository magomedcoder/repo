package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type AdminHandler struct {
	admin     *usecase.AdminUseCase
	accessUC  *usecase.AccessLogUseCase
	mirrors   *usecase.AdminMirrorUseCase
	accessLog *audit.Logger
}

func NewAdminHandler(admin *usecase.AdminUseCase, accessUC *usecase.AccessLogUseCase) *AdminHandler {
	return &AdminHandler{
		admin:    admin,
		accessUC: accessUC,
	}
}

func (h *AdminHandler) SetAccessLogger(log *audit.Logger) {
	h.accessLog = log
}

func (h *AdminHandler) SetMirrors(mirrors *usecase.AdminMirrorUseCase) {
	h.mirrors = mirrors
}

type adminCreateRequest struct {
	Username        string      `json:"username"`
	Email           string      `json:"email"`
	Password        string      `json:"password"`
	Role            domain.Role `json:"role"`
	CurrentPassword string      `json:"current_password"`
}

type adminUpdateRequest struct {
	Email           *string      `json:"email"`
	Role            *domain.Role `json:"role"`
	Password        *string      `json:"password"`
	CurrentPassword string       `json:"current_password"`
}

type adminDeleteRequest struct {
	CurrentPassword string `json:"current_password"`
}

func (h *AdminHandler) Overview(c *httpx.Context) error {
	actor, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.admin.Overview(c.Ctx(), actor.ID)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) ListUsers(c *httpx.Context) error {
	actor, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	var role *domain.Role
	if raw := strings.TrimSpace(c.Query("role")); raw != "" {
		if parsed, ok := domain.ParseRole(raw); ok {
			role = &parsed
		}
	}

	out, err := h.admin.ListUsers(c.Ctx(), actor.ID, usecase.AdminListInput{
		Query:  c.Query("q"),
		Role:   role,
		Sort:   c.Query("sort"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) ListRepos(c *httpx.Context) error {
	actor, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	filter := domain.AdminRepoFilter{
		Query:  c.Query("q"),
		Order:  domain.RepoOrder(c.Query("sort")),
		Limit:  limit,
		Offset: offset,
		Mirror: c.Query("mirror") == "1" || c.Query("mirror") == "true",
	}

	if raw := c.Query("private"); raw == "1" || raw == "true" {
		v := true
		filter.Private = &v
	} else if raw == "0" || raw == "false" {
		v := false
		filter.Private = &v
	}

	out, err := h.admin.ListRepos(c.Ctx(), actor.ID, filter)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) ListAccessLogs(c *httpx.Context) error {
	if h.accessUC == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	actor, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	filter := domain.AccessLogFilter{
		Action:   c.Query("action"),
		IP:       c.Query("ip"),
		Username: c.Query("username"),
		Limit:    limit,
		Offset:   offset,
	}

	if raw := c.Query("from"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.Since = &t
		}
	}

	if raw := c.Query("to"); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.Until = &t
		}
	}

	out, err := h.accessUC.ListFilter(c.Ctx(), actor.ID, filter)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) CreateUser(c *httpx.Context) error {
	actor, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[adminCreateRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.admin.Create(c.Ctx(), actor.ID, usecase.AdminCreateInput{
		Username:        req.Username,
		Email:           req.Email,
		Password:        req.Password,
		Role:            req.Role,
		CurrentPassword: req.CurrentPassword,
	})
	if err != nil {
		return mapError(c, err)
	}

	if h.accessLog != nil {
		h.accessLog.AdminUserCreate(c.R, actor, out.Username)
	}

	return httpx.Created(c, out)
}

func (h *AdminHandler) UpdateUser(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return c.Error(http.StatusBadRequest, "invalid_user_id")
	}

	req, err := httpx.BindJSON[adminUpdateRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.admin.Update(c.Ctx(), user.ID, uint(id), usecase.AdminUpdateInput{
		Email:           req.Email,
		Role:            req.Role,
		Password:        req.Password,
		CurrentPassword: req.CurrentPassword,
	})
	if err != nil {
		return mapError(c, err)
	}

	if h.accessLog != nil {
		detail := strconv.FormatUint(id, 10)
		if req.Role != nil {
			detail += ":role"
		}

		if req.Password != nil {
			detail += ":password"
		}

		if req.Email != nil {
			detail += ":email"
		}

		h.accessLog.AdminUserUpdate(c.R, user, detail)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) DeleteUser(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return c.Error(http.StatusBadRequest, "invalid_user_id")
	}

	var currentPassword string
	if req, bindErr := httpx.BindJSON[adminDeleteRequest](c); bindErr == nil {
		currentPassword = req.CurrentPassword
	}

	if err := h.admin.Delete(c.Ctx(), user.ID, uint(id), currentPassword); err != nil {
		return mapError(c, err)
	}

	if h.accessLog != nil {
		h.accessLog.AdminUserDelete(c.R, user, strconv.FormatUint(id, 10))
	}
	return c.NoContent()
}
