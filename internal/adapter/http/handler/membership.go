package handler

import (
	"net/http"

	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

type MembershipHandler struct {
	access *usecase.AccessUseCase
}

func NewMembershipHandler(access *usecase.AccessUseCase) *MembershipHandler {
	return &MembershipHandler{access: access}
}

type collaboratorRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *MembershipHandler) ListCollaborators(c *httpx.Context, in usecase.ResolveRepositoryInput) error {
	out, err := h.access.ListCollaborators(c.Ctx(), in)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"collaborators": out})
}

func (h *MembershipHandler) AddCollaborator(c *httpx.Context, in usecase.ResolveRepositoryInput) error {
	req, err := httpx.BindJSON[collaboratorRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.access.AddCollaborator(c.Ctx(), usecase.UpsertCollaboratorInput{
		ResolveRepositoryInput: in,
		Username:               req.Username,
		Role:                   req.Role,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *MembershipHandler) RemoveCollaborator(c *httpx.Context, in usecase.ResolveRepositoryInput, username string) error {
	if err := h.access.RemoveCollaborator(c.Ctx(), in, username); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}
