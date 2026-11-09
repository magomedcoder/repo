package handler

import (
	"net/http"

	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

type PullRequestHandler struct {
	pulls *usecase.PullRequestUseCase
}

func NewPullRequestHandler(pulls *usecase.PullRequestUseCase) *PullRequestHandler {
	return &PullRequestHandler{pulls: pulls}
}

type pullWriteRequest struct {
	Title      *string `json:"title"`
	Body       *string `json:"body"`
	State      *string `json:"state"`
	BaseBranch *string `json:"base_branch"`
	HeadBranch *string `json:"head_branch"`
}

type pullMergeRequest struct {
	Strategy string `json:"strategy"`
}

func (h *PullRequestHandler) get(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "pulls":
		offset, limit := usecase.ParseOffsetLimit(c.Query("offset"), c.Query("limit"))
		items, err := h.pulls.List(c.Ctx(), in, c.Query("state"), offset, limit)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, pullsResponse{Pulls: items})
	case "pull":
		number, err := requireExtraPositive(c, extra, "pull_not_found")
		if err != nil {
			return err
		}

		item, err := h.pulls.Get(c.Ctx(), in, number)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	case "pull-diff":
		number, err := requireExtraPositive(c, extra, "pull_not_found")
		if err != nil {
			return err
		}

		item, err := h.pulls.Diff(c.Ctx(), in, number)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (h *PullRequestHandler) post(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "pulls":
		req, err := bindJSON[pullWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.pulls.Create(c.Ctx(), in, derefStr(req.Title), derefStr(req.Body), derefStr(req.BaseBranch), derefStr(req.HeadBranch))
		if err != nil {
			return mapError(c, err)
		}

		return httpx.Created(c, item)
	case "pull-comments":
		number, err := requireExtraPositive(c, extra, "pull_not_found")
		if err != nil {
			return err
		}

		req, err := bindJSON[commentWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.pulls.AddComment(c.Ctx(), in, number, req.Body)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.Created(c, item)
	case "pull-merge":
		number, err := requireExtraPositive(c, extra, "pull_not_found")
		if err != nil {
			return err
		}

		req, err := bindJSON[pullMergeRequest](c)
		if err != nil {
			return err
		}

		item, err := h.pulls.Merge(c.Ctx(), in, number, req.Strategy)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (h *PullRequestHandler) patch(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "pull":
		number, err := requireExtraPositive(c, extra, "pull_not_found")
		if err != nil {
			return err
		}

		req, err := bindJSON[pullWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.pulls.Update(c.Ctx(), in, number, req.Title, req.Body, req.State, req.BaseBranch, req.HeadBranch)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	case "pull-comment":
		number, commentID, err := requireCommentIDs(c, extra)
		if err != nil {
			return err
		}

		req, err := bindJSON[commentWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.pulls.UpdateComment(c.Ctx(), in, number, uint(commentID), req.Body)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (h *PullRequestHandler) delete(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "pull":
		number, err := requireExtraPositive(c, extra, "pull_not_found")
		if err != nil {
			return err
		}

		if err := h.pulls.Delete(c.Ctx(), in, number); err != nil {
			return mapError(c, err)
		}

		return c.NoContent()
	case "pull-comment":
		number, commentID, err := requireCommentIDs(c, extra)
		if err != nil {
			return err
		}

		if err := h.pulls.DeleteComment(c.Ctx(), in, number, uint(commentID)); err != nil {
			return mapError(c, err)
		}

		return c.NoContent()
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func splitPullPath(parts []string) (action string, repoParts, extra []string, ok bool) {
	n := len(parts)
	if n >= 3 && parts[n-3] == "pulls" {
		switch parts[n-1] {
		case "merge":
			return "pull-merge", parts[:n-3], []string{parts[n-2]}, true
		case "diff":
			return "pull-diff", parts[:n-3], []string{parts[n-2]}, true
		}
	}

	return splitNamedResource(parts, "pulls", "pull")
}
