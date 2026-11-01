package handler

import (
	"net/http"
	"strconv"

	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type AdminHandler struct {
	admin     *usecase.AdminUseCase
	accessUC  *usecase.AccessLogUseCase
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

type adminCreateRequest struct {
	Username string      `json:"username"`
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
}

type adminUpdateRequest struct {
	Email    *string      `json:"email"`
	Role     *domain.Role `json:"role"`
	Password *string      `json:"password"`
}

func (h *AdminHandler) ListUsers(c *httpx.Context) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	out, err := h.admin.List(c.Ctx(), c.Query("q"), limit, offset)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) ListAccessLogs(c *httpx.Context) error {
	if h.accessUC == nil {
		return c.Error(http.StatusNotFound, "not_found")
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	out, err := h.accessUC.List(c.Ctx(), c.Query("action"), c.Query("ip"), c.Query("username"), limit, offset)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) GetUser(c *httpx.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return c.Error(http.StatusBadRequest, "invalid_user_id")
	}

	out, err := h.admin.Get(c.Ctx(), uint(id))
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *AdminHandler) CreateUser(c *httpx.Context) error {
	actor, _ := middleware.UserFromContext(c.Ctx())
	req, err := httpx.BindJSON[adminCreateRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.admin.Create(c.Ctx(), usecase.AdminCreateInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
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
		Email:    req.Email,
		Role:     req.Role,
		Password: req.Password,
	})
	if err != nil {
		return mapError(c, err)
	}

	if h.accessLog != nil {
		h.accessLog.AdminUserUpdate(c.R, user, strconv.FormatUint(id, 10))
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

	if err := h.admin.Delete(c.Ctx(), user.ID, uint(id)); err != nil {
		return mapError(c, err)
	}

	if h.accessLog != nil {
		h.accessLog.AdminUserDelete(c.R, user, strconv.FormatUint(id, 10))
	}

	return c.NoContent()
}
