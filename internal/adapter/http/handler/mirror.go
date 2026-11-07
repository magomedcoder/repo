package handler

import (
	"io"
	"strconv"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type checkMirrorRequest struct {
	URL      string  `json:"url"`
	Username *string `json:"username"`
	Password *string `json:"password"`
	DryRun   bool    `json:"dry_run"`
}

func (h *RepositoryHandler) mirrorInput(c *httpx.Context) (usecase.ResolveRepositoryInput, error) {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return usecase.ResolveRepositoryInput{}, usecase.ErrUnauthorized
	}

	name, err := parseRepoName(c)
	if err != nil {
		return usecase.ResolveRepositoryInput{}, usecase.ErrRepoNotFound
	}

	return usecase.ResolveRepositoryInput{
		Name:     name,
		ViewerID: user.ID,
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

func (h *RepositoryHandler) ExportMirrorBundle(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	c.W.Header().Set("Content-Type", "application/octet-stream")
	c.W.Header().Set("Content-Disposition", `attachment; filename="mirror.bundle"`)

	if err := h.repos.ExportMirrorBundle(c.Ctx(), usecase.BundleInput{
		ResolveRepositoryInput: in,
		CurrentPassword:        bundleStepUpPassword(c),
	}, c.W); err != nil {
		return mapError(c, err)
	}

	return nil
}

func (h *RepositoryHandler) ImportMirrorBundle(c *httpx.Context) error {
	in, err := h.mirrorInput(c)
	if err != nil {
		return mapError(c, err)
	}

	if err := h.repos.ImportMirrorBundle(c.Ctx(), usecase.BundleInput{
		ResolveRepositoryInput: in,
		CurrentPassword:        bundleStepUpPassword(c),
	}, io.LimitReader(c.R.Body, 2<<30)); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func bundleStepUpPassword(c *httpx.Context) string {
	return strings.TrimSpace(c.R.Header.Get("X-Repo-Password"))
}
