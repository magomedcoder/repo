package handler

import (
	"net/http"
	"strconv"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type AdminHandler struct {
	admin *usecase.AdminUseCase
}

func NewAdminHandler(admin *usecase.AdminUseCase) *AdminHandler {
	return &AdminHandler{
		admin: admin,
	}
}

type adminCreateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type adminUpdateRequest struct {
	Email    *string `json:"email"`
	Role     *string `json:"role"`
	Password *string `json:"password"`
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

	return c.NoContent()
}
