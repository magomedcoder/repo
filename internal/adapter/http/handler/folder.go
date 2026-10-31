package handler

import (
	"net/http"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type FolderHandler struct {
	folders *usecase.FolderUseCase
}

func NewFolderHandler(folders *usecase.FolderUseCase) *FolderHandler {
	return &FolderHandler{folders: folders}
}

type createFolderRequest struct {
	Name string `json:"name"`
}

type renameFolderRequest struct {
	Name string `json:"name"`
}

func (h *FolderHandler) Create(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[createFolderRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.folders.Create(c.Ctx(), usecase.CreateFolderInput{
		ViewerID: user.ID,
		OwnerID:  user.ID,
		Name:     req.Name,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *FolderHandler) List(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.folders.List(c.Ctx(), user.ID, user.ID)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, foldersResponse{Folders: out})
}

func (h *FolderHandler) Get(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path := strings.Trim(c.Param("path"), "/")
	out, err := h.folders.GetContents(c.Ctx(), user.ID, user.ID, path)
	if err != nil {
		return mapError(c, err)
	}
	return httpx.OK(c, out)
}

func (h *FolderHandler) Rename(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path := strings.Trim(c.Param("path"), "/")
	req, err := httpx.BindJSON[renameFolderRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.folders.Rename(c.Ctx(), usecase.RenameFolderInput{
		ViewerID: user.ID,
		OwnerID:  user.ID,
		Path:     path,
		Name:     req.Name,
	})

	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *FolderHandler) Delete(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path := strings.Trim(c.Param("path"), "/")
	if err := h.folders.Delete(c.Ctx(), user.ID, user.ID, path); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}
