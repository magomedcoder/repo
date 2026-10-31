package handler

import (
	"net/http"

	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
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
		number, ok := parsePositive(extra[0])
		if !ok {
			return c.Error(http.StatusBadRequest, "pull_not_found")
		}

		item, err := h.pulls.Get(c.Ctx(), in, number)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	case "pull-diff":
		number, ok := parsePositive(extra[0])
		if !ok {
			return c.Error(http.StatusBadRequest, "pull_not_found")
		}

		item, err := h.pulls.Diff(c.Ctx(), in, number)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	case "pull-commits":
		number, ok := parsePositive(extra[0])
		if !ok {
			return c.Error(http.StatusBadRequest, "pull_not_found")
		}

		items, err := h.pulls.Commits(c.Ctx(), in, number)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, commitsResponse{Commits: items})
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (h *PullRequestHandler) post(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "pulls":
		req, err := httpx.BindJSON[pullWriteRequest](c)
		if err != nil {
			return c.Error(http.StatusBadRequest, "invalid_json")
		}

		title, body, base, head := "", "", "", ""
		if req.Title != nil {
			title = *req.Title
		}

		if req.Body != nil {
			body = *req.Body
		}

		if req.BaseBranch != nil {
			base = *req.BaseBranch
		}

		if req.HeadBranch != nil {
			head = *req.HeadBranch
		}

		item, err := h.pulls.Create(c.Ctx(), in, title, body, base, head)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.Created(c, item)
	case "pull-comments":
		number, ok := parsePositive(extra[0])
		if !ok {
			return c.Error(http.StatusBadRequest, "pull_not_found")
		}

		req, err := httpx.BindJSON[commentWriteRequest](c)
		if err != nil {
			return c.Error(http.StatusBadRequest, "invalid_json")
		}

		item, err := h.pulls.AddComment(c.Ctx(), in, number, req.Body)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.Created(c, item)
	case "pull-merge":
		number, ok := parsePositive(extra[0])
		if !ok {
			return c.Error(http.StatusBadRequest, "pull_not_found")
		}

		req, _ := httpx.BindJSON[pullMergeRequest](c)
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
		number, ok := parsePositive(extra[0])
		if !ok {
			return c.Error(http.StatusBadRequest, "pull_not_found")
		}

		req, err := httpx.BindJSON[pullWriteRequest](c)
		if err != nil {
			return c.Error(http.StatusBadRequest, "invalid_json")
		}

		item, err := h.pulls.Update(c.Ctx(), in, number, req.Title, req.Body, req.State, req.BaseBranch, req.HeadBranch)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	case "pull-comment":
		number, ok := parsePositive(extra[0])
		commentID, okID := parsePositive(extra[1])
		if !ok || !okID {
			return c.Error(http.StatusBadRequest, "comment_not_found")
		}

		req, err := httpx.BindJSON[commentWriteRequest](c)
		if err != nil {
			return c.Error(http.StatusBadRequest, "invalid_json")
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
		number, ok := parsePositive(extra[0])
		if !ok {
			return c.Error(http.StatusBadRequest, "pull_not_found")
		}

		if err := h.pulls.Delete(c.Ctx(), in, number); err != nil {
			return mapError(c, err)
		}

		return c.NoContent()
	case "pull-comment":
		number, ok := parsePositive(extra[0])
		commentID, okID := parsePositive(extra[1])
		if !ok || !okID {
			return c.Error(http.StatusBadRequest, "comment_not_found")
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
	if n < 2 {
		return "", nil, nil, false
	}

	if n >= 4 && parts[n-2] == "comments" && parts[n-4] == "pulls" {
		return "pull-comment", parts[:n-4], []string{parts[n-3], parts[n-1]}, true
	}

	if n >= 3 && parts[n-1] == "comments" && parts[n-3] == "pulls" {
		return "pull-comments", parts[:n-3], []string{parts[n-2]}, true
	}

	if n >= 3 && parts[n-1] == "merge" && parts[n-3] == "pulls" {
		return "pull-merge", parts[:n-3], []string{parts[n-2]}, true
	}

	if n >= 3 && parts[n-1] == "diff" && parts[n-3] == "pulls" {
		return "pull-diff", parts[:n-3], []string{parts[n-2]}, true
	}

	if n >= 3 && parts[n-1] == "commits" && parts[n-3] == "pulls" {
		return "pull-commits", parts[:n-3], []string{parts[n-2]}, true
	}

	if parts[n-1] == "pulls" {
		return "pulls", parts[:n-1], nil, true
	}

	if parts[n-2] == "pulls" {
		return "pull", parts[:n-2], []string{parts[n-1]}, true
	}

	return "", nil, nil, false
}
