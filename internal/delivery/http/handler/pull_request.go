package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
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

func (h *PullRequestHandler) Handle(w http.ResponseWriter, r *http.Request) bool {
	owner := strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		return false
	}

	action, repoParts, extra, ok := splitPullPath(strings.Split(rest, "/"))
	if !ok {
		return false
	}

	folderPath, name, err := folderAndName(repoParts)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_repository_path")
		return true
	}

	var viewerID uint
	if user, loggedIn := middleware.UserFromContext(r.Context()); loggedIn {
		viewerID = user.ID
	}

	in := usecase.ResolveRepositoryInput{
		OwnerUsername: owner,
		FolderPath:    folderPath,
		Name:          name,
		ViewerID:      viewerID,
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r, in, action, extra)
	case http.MethodPost:
		h.post(w, r, in, action, extra)
	case http.MethodPatch:
		h.patch(w, r, in, action, extra)
	case http.MethodDelete:
		h.delete(w, r, in, action, extra)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
	return true
}

func (h *PullRequestHandler) get(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "pulls":
		offset, limit := usecase.ParseOffsetLimit(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
		items, err := h.pulls.List(in, r.URL.Query().Get("state"), offset, limit)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pulls": items})
	case "pull":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "pull_not_found")
			return
		}

		item, err := h.pulls.Get(in, number)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case "pull-diff":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "pull_not_found")
			return
		}

		item, err := h.pulls.Diff(in, number)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case "pull-commits":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "pull_not_found")
			return
		}

		items, err := h.pulls.Commits(in, number)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"commits": items})
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (h *PullRequestHandler) post(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "pulls":
		var req pullWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
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

		item, err := h.pulls.Create(in, title, body, base, head)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusCreated, item)
	case "pull-comments":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "pull_not_found")
			return
		}

		var req commentWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		item, err := h.pulls.AddComment(in, number, req.Body)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusCreated, item)
	case "pull-merge":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "pull_not_found")
			return
		}

		var req pullMergeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		item, err := h.pulls.Merge(in, number, req.Strategy)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (h *PullRequestHandler) patch(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "pull":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "pull_not_found")
			return
		}

		var req pullWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		item, err := h.pulls.Update(in, number, req.Title, req.Body, req.State, req.BaseBranch, req.HeadBranch)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case "pull-comment":
		number, ok := parsePositive(extra[0])
		commentID, okID := parsePositive(extra[1])
		if !ok || !okID {
			writeError(w, r, http.StatusBadRequest, "comment_not_found")
			return
		}

		var req commentWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		item, err := h.pulls.UpdateComment(in, number, uint(commentID), req.Body)
		if err != nil {
			writePullError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (h *PullRequestHandler) delete(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "pull":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "pull_not_found")
			return
		}

		if err := h.pulls.Delete(in, number); err != nil {
			writePullError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	case "pull-comment":
		number, ok := parsePositive(extra[0])
		commentID, okID := parsePositive(extra[1])
		if !ok || !okID {
			writeError(w, r, http.StatusBadRequest, "comment_not_found")
			return
		}

		if err := h.pulls.DeleteComment(in, number, uint(commentID)); err != nil {
			writePullError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
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

func writePullError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrPullTitleRequired),
		errors.Is(err, usecase.ErrInvalidPullState),
		errors.Is(err, usecase.ErrSameBranch),
		errors.Is(err, usecase.ErrInvalidMergeStrategy),
		errors.Is(err, usecase.ErrCommentBodyRequired),
		errors.Is(err, usecase.ErrPullNotOpen):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrBranchNotFound),
		errors.Is(err, usecase.ErrPullNotFound),
		errors.Is(err, usecase.ErrCommentNotFound),
		errors.Is(err, usecase.ErrRepoNotFound),
		errors.Is(err, usecase.ErrFolderNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrMergeConflict),
		errors.Is(err, usecase.ErrNotFastForward):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrPullForbidden),
		errors.Is(err, usecase.ErrRepoForbidden):
		writeError(w, r, http.StatusForbidden, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
