package handler

import (
	"net/http"

	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type IssueHandler struct {
	issues *usecase.IssueUseCase
}

func NewIssueHandler(issues *usecase.IssueUseCase) *IssueHandler {
	return &IssueHandler{issues: issues}
}

type issueWriteRequest struct {
	Title    *string `json:"title"`
	Body     *string `json:"body"`
	State    *string `json:"state"`
	LabelIDs *[]uint `json:"label_ids"`
}

type labelWriteRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (h *IssueHandler) get(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "issues":
		offset, limit := usecase.ParseOffsetLimit(c.Query("offset"), c.Query("limit"))
		items, err := h.issues.List(c.Ctx(), in, c.Query("state"), offset, limit)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, issuesResponse{Issues: items})
	case "issue":
		number, err := requireExtraPositive(c, extra, "issue_not_found")
		if err != nil {
			return err
		}

		item, err := h.issues.Get(c.Ctx(), in, number)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	case "labels":
		items, err := h.issues.ListLabels(c.Ctx(), in)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, labelsResponse{Labels: items})
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (h *IssueHandler) post(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "issues":
		req, err := bindJSON[issueWriteRequest](c)
		if err != nil {
			return err
		}

		var labels []uint
		if req.LabelIDs != nil {
			labels = *req.LabelIDs
		}

		item, err := h.issues.Create(c.Ctx(), in, derefStr(req.Title), derefStr(req.Body), labels)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.Created(c, item)
	case "issue-comments":
		number, err := requireExtraPositive(c, extra, "issue_not_found")
		if err != nil {
			return err
		}

		req, err := bindJSON[commentWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.issues.AddComment(c.Ctx(), in, number, req.Body)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.Created(c, item)
	case "labels":
		req, err := bindJSON[labelWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.issues.CreateLabel(c.Ctx(), in, derefStr(req.Name), derefStr(req.Color))
		if err != nil {
			return mapError(c, err)
		}

		return httpx.Created(c, item)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (h *IssueHandler) patch(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "issue":
		number, err := requireExtraPositive(c, extra, "issue_not_found")
		if err != nil {
			return err
		}

		req, err := bindJSON[issueWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.issues.Update(c.Ctx(), in, number, req.Title, req.Body, req.State, req.LabelIDs)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	case "issue-comment":
		number, commentID, err := requireCommentIDs(c, extra)
		if err != nil {
			return err
		}

		req, err := bindJSON[commentWriteRequest](c)
		if err != nil {
			return err
		}

		item, err := h.issues.UpdateComment(c.Ctx(), in, number, commentID, req.Body)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (h *IssueHandler) delete(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	switch action {
	case "issue":
		number, err := requireExtraPositive(c, extra, "issue_not_found")
		if err != nil {
			return err
		}

		if err := h.issues.Delete(c.Ctx(), in, number); err != nil {
			return mapError(c, err)
		}

		return c.NoContent()
	case "issue-comment":
		number, commentID, err := requireCommentIDs(c, extra)
		if err != nil {
			return err
		}

		if err := h.issues.DeleteComment(c.Ctx(), in, number, commentID); err != nil {
			return mapError(c, err)
		}

		return c.NoContent()
	case "label":
		id, err := requireExtraPositive(c, extra, "label_not_found")
		if err != nil {
			return err
		}

		if err := h.issues.DeleteLabel(c.Ctx(), in, uint(id)); err != nil {
			return mapError(c, err)
		}

		return c.NoContent()
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func splitIssuePath(parts []string) (action string, repoParts, extra []string, ok bool) {
	if a, rp, ex, ok := splitNamedResource(parts, "issues", "issue"); ok {
		return a, rp, ex, true
	}

	n := len(parts)
	if n < 2 {
		return "", nil, nil, false
	}

	if parts[n-1] == "labels" {
		return "labels", parts[:n-1], nil, true
	}

	if parts[n-2] == "labels" {
		return "label", parts[:n-2], []string{parts[n-1]}, true
	}

	return "", nil, nil, false
}
