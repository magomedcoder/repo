package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
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

type commentWriteRequest struct {
	Body string `json:"body"`
}

type labelWriteRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (h *IssueHandler) Handle(w http.ResponseWriter, r *http.Request) bool {
	owner := strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		return false
	}

	action, repoParts, extra, ok := splitIssuePath(strings.Split(rest, "/"))
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

func (h *IssueHandler) get(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "issues":
		offset, limit := usecase.ParseOffsetLimit(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
		items, err := h.issues.List(in, r.URL.Query().Get("state"), offset, limit)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"issues": items})
	case "issue":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "issue_not_found")
			return
		}

		item, err := h.issues.Get(in, number)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, item)
	case "labels":
		items, err := h.issues.ListLabels(in)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"labels": items})
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (h *IssueHandler) post(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "issues":
		var req issueWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		title := ""
		body := ""
		if req.Title != nil {
			title = *req.Title
		}

		if req.Body != nil {
			body = *req.Body
		}

		var labels []uint
		if req.LabelIDs != nil {
			labels = *req.LabelIDs
		}

		item, err := h.issues.Create(in, title, body, labels)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	case "issue-comments":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "issue_not_found")
			return
		}

		var req commentWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		item, err := h.issues.AddComment(in, number, req.Body)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	case "labels":
		var req labelWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		name, color := "", ""
		if req.Name != nil {
			name = *req.Name
		}

		if req.Color != nil {
			color = *req.Color
		}

		item, err := h.issues.CreateLabel(in, name, color)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (h *IssueHandler) patch(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "issue":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "issue_not_found")
			return
		}

		var req issueWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		item, err := h.issues.Update(in, number, req.Title, req.Body, req.State, req.LabelIDs)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case "issue-comment":
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

		item, err := h.issues.UpdateComment(in, number, commentID, req.Body)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case "label":
		id, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "label_not_found")
			return
		}

		var req labelWriteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_json")
			return
		}

		item, err := h.issues.UpdateLabel(in, uint(id), req.Name, req.Color)
		if err != nil {
			writeIssueError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (h *IssueHandler) delete(w http.ResponseWriter, r *http.Request, in usecase.ResolveRepositoryInput, action string, extra []string) {
	switch action {
	case "issue":
		number, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "issue_not_found")
			return
		}

		if err := h.issues.Delete(in, number); err != nil {
			writeIssueError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case "issue-comment":
		number, ok := parsePositive(extra[0])
		commentID, okID := parsePositive(extra[1])
		if !ok || !okID {
			writeError(w, r, http.StatusBadRequest, "comment_not_found")
			return
		}

		if err := h.issues.DeleteComment(in, number, commentID); err != nil {
			writeIssueError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case "label":
		id, ok := parsePositive(extra[0])
		if !ok {
			writeError(w, r, http.StatusBadRequest, "label_not_found")
			return
		}

		if err := h.issues.DeleteLabel(in, uint(id)); err != nil {
			writeIssueError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func splitIssuePath(parts []string) (action string, repoParts, extra []string, ok bool) {
	n := len(parts)
	if n < 2 {
		return "", nil, nil, false
	}

	if n >= 4 && parts[n-2] == "comments" && parts[n-4] == "issues" {
		return "issue-comment", parts[:n-4], []string{parts[n-3], parts[n-1]}, true
	}

	if n >= 3 && parts[n-1] == "comments" && parts[n-3] == "issues" {
		return "issue-comments", parts[:n-3], []string{parts[n-2]}, true
	}

	if parts[n-1] == "issues" {
		return "issues", parts[:n-1], nil, true
	}

	if parts[n-2] == "issues" {
		return "issue", parts[:n-2], []string{parts[n-1]}, true
	}

	if parts[n-1] == "labels" {
		return "labels", parts[:n-1], nil, true
	}

	if parts[n-2] == "labels" {
		return "label", parts[:n-2], []string{parts[n-1]}, true
	}
	return "", nil, nil, false
}

func parsePositive(raw string) (int, bool) {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, false
	}
	
	return n, true
}

func writeIssueError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrIssueTitleRequired),
		errors.Is(err, usecase.ErrCommentBodyRequired),
		errors.Is(err, usecase.ErrLabelNameRequired),
		errors.Is(err, usecase.ErrInvalidIssueState),
		errors.Is(err, usecase.ErrInvalidLabel),
		errors.Is(err, usecase.ErrInvalidName):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrLabelExists):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrIssueNotFound),
		errors.Is(err, usecase.ErrCommentNotFound),
		errors.Is(err, usecase.ErrLabelNotFound),
		errors.Is(err, usecase.ErrRepoNotFound),
		errors.Is(err, usecase.ErrFolderNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrIssueForbidden),
		errors.Is(err, usecase.ErrRepoForbidden):
		writeError(w, r, http.StatusForbidden, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
