package handler

import (
	"bytes"
	"io"
	"net/http"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type ProfileHandler struct {
	profiles *usecase.ProfileUseCase
}

func NewProfileHandler(profiles *usecase.ProfileUseCase) *ProfileHandler {
	return &ProfileHandler{
		profiles: profiles,
	}
}

type updateProfileRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"current_password"`
}

func (h *ProfileHandler) Get(c *httpx.Context) error {
	username := c.Param("username")
	var viewerID uint
	if user, ok := middleware.UserFromContext(c.Ctx()); ok {
		viewerID = user.ID
	}

	out, err := h.profiles.GetByUsername(c.Ctx(), username, viewerID)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *ProfileHandler) UpdateMe(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[updateProfileRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.profiles.UpdateEmail(c.Ctx(), user.ID, req.Email, req.CurrentPassword)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *ProfileHandler) UploadAvatar(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	if err := c.R.ParseMultipartForm(2 << 20); err != nil {
		return c.Error(http.StatusBadRequest, "invalid_avatar")
	}
	defer func() {
		if c.R.MultipartForm != nil {
			_ = c.R.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := c.R.FormFile("avatar")
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_avatar")
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	reader := io.MultiReader(bytes.NewReader(buf[:n]), file)

	out, err := h.profiles.SetAvatar(c.Ctx(), user.ID, reader, contentType, header.Size)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *ProfileHandler) DeleteAvatar(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.profiles.ClearAvatar(c.Ctx(), user.ID)
	if err != nil {
		return mapError(c, err)
	}
	return httpx.OK(c, out)
}

func (h *ProfileHandler) ServeAvatar(c *httpx.Context) error {
	username := c.Param("username")
	f, contentType, err := h.profiles.OpenAvatar(c.Ctx(), username)
	if err != nil {
		return mapError(c, err)
	}

	defer f.Close()
	c.W.Header().Set("Content-Type", contentType)
	c.W.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = io.Copy(c.W, f)
	return nil
}
