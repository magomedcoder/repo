package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

func (h *RepositoryHandler) DispatchGet(w http.ResponseWriter, r *http.Request) {
	owner := strings.TrimSpace(r.PathValue("owner"))
	rest := strings.Trim(r.PathValue("path"), "/")
	if owner == "" || rest == "" {
		writeError(w, http.StatusBadRequest, "invalid repository path")
		return
	}

	parts := strings.Split(rest, "/")
	action, repoParts, extra, ok := splitBrowsePath(parts)
	if !ok {
		h.Get(w, r)
		return
	}

	folderPath, name, err := folderAndName(repoParts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
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

	ref := r.URL.Query().Get("ref")
	path := r.URL.Query().Get("path")

	switch action {
	case "branches":
		items, err := h.repos.ListBranches(in)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"branches": items})

	case "tags":
		items, err := h.repos.ListTags(in)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"tags": items})

	case "commits":
		if len(extra) == 0 {
			offset, limit := usecase.ParseOffsetLimit(r.URL.Query().Get("offset"), r.URL.Query().Get("limit"))
			items, err := h.repos.ListCommits(in, ref, offset, limit)
			if err != nil {
				writeRepoError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{"commits": items})
			return
		}
		sha := extra[0]
		if len(extra) == 1 {
			item, err := h.repos.GetCommit(in, sha)
			if err != nil {
				writeRepoError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, item)
			return
		}
		if len(extra) == 2 && extra[1] == "diff" {
			item, err := h.repos.GetCommitDiff(in, sha)
			if err != nil {
				writeRepoError(w, err)
				return
			}

			writeJSON(w, http.StatusOK, item)
			return
		}
		writeError(w, http.StatusNotFound, "not found")

	case "tree":
		items, err := h.repos.ListTree(in, ref, path)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ref": ref, "path": path, "tree": items})

	case "blob":
		item, err := h.repos.GetBlob(in, ref, path)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)

	case "raw":
		item, err := h.repos.GetBlob(in, ref, path)
		if err != nil {
			writeRepoError(w, err)
			return
		}
		var data []byte
		if item.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(item.Content)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			data = decoded
		} else {
			data = []byte(item.Content)
		}

		if item.IsBinary {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)

	case "readme":
		item, err := h.repos.GetReadme(in, ref)
		if err != nil {
			writeRepoError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, item)

	case "stats":
		item, err := h.repos.GetStats(in, ref)
		if err != nil {
			writeRepoError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, item)

	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func splitBrowsePath(parts []string) (action string, repoParts, extra []string, ok bool) {
	if len(parts) < 2 {
		return "", nil, nil, false
	}

	// .../commits/{sha}/diff
	if len(parts) >= 3 && parts[len(parts)-1] == "diff" && parts[len(parts)-3] == "commits" {
		return "commits", parts[:len(parts)-3], parts[len(parts)-2:], true
	}

	// .../commits/{sha}
	if len(parts) >= 2 && parts[len(parts)-2] == "commits" {
		return "commits", parts[:len(parts)-2], parts[len(parts)-1:], true
	}

	last := parts[len(parts)-1]
	switch last {
	case "branches", "tags", "commits", "tree", "blob", "raw", "readme", "stats":
		return last, parts[:len(parts)-1], nil, true
	default:
		return "", nil, nil, false
	}
}

func folderAndName(parts []string) (folderPath, name string, err error) {
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "", "", errors.New("invalid repository path")
	}

	name = parts[len(parts)-1]
	if len(parts) > 1 {
		folderPath = strings.Join(parts[:len(parts)-1], "/")
	}

	return folderPath, name, nil
}
