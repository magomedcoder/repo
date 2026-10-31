package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

func (h *RepositoryHandler) serveBrowseGet(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	ref := c.Query("ref")
	path := c.Query("path")

	switch action {
	case "branches":
		items, err := h.repos.ListBranches(c.Ctx(), in)
		if err != nil {
			return mapError(c, err)
		}
		return httpx.OK(c, branchesResponse{Branches: items})

	case "tags":
		items, err := h.repos.ListTags(c.Ctx(), in)
		if err != nil {
			return mapError(c, err)
		}
		return httpx.OK(c, tagsResponse{Tags: items})

	case "commits":
		if len(extra) == 0 {
			offset, limit := usecase.ParseOffsetLimit(c.Query("offset"), c.Query("limit"))
			if limit <= 0 {
				limit = 30
			}
			if limit > 100 {
				limit = 100
			}
			items, err := h.repos.ListCommits(c.Ctx(), in, ref, offset, limit)
			if err != nil {
				return mapError(c, err)
			}

			return httpx.OK(c, commitsResponse{Commits: items})
		}
		sha := extra[0]
		if len(extra) == 1 {
			item, err := h.repos.GetCommit(c.Ctx(), in, sha)
			if err != nil {
				return mapError(c, err)
			}

			return httpx.OK(c, item)
		}
		if len(extra) == 2 && extra[1] == "diff" {
			item, err := h.repos.GetCommitDiff(c.Ctx(), in, sha)
			if err != nil {
				return mapError(c, err)
			}

			return httpx.OK(c, item)
		}
		return c.Error(http.StatusNotFound, "not_found")

	case "tree":
		items, err := h.repos.ListTree(c.Ctx(), in, ref, path)
		if err != nil {
			return mapError(c, err)
		}
		return httpx.OK(c, treeResponse{Ref: ref, Path: path, Tree: items})

	case "blob":
		item, err := h.repos.GetBlob(c.Ctx(), in, ref, path)
		if err != nil {
			return mapError(c, err)
		}
		return httpx.OK(c, item)

	case "blame":
		item, err := h.repos.GetBlame(c.Ctx(), in, ref, path)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)

	case "raw":
		item, err := h.repos.GetBlob(c.Ctx(), in, ref, path)
		if err != nil {
			return mapError(c, err)
		}
		var data []byte
		if item.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(item.Content)
			if err != nil {
				return c.Error(http.StatusInternalServerError, "internal_error")
			}
			data = decoded
		} else {
			data = []byte(item.Content)
		}

		if item.IsBinary {
			c.W.Header().Set("Content-Type", "application/octet-stream")
		} else {
			c.W.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}

		c.W.Header().Set("Content-Length", strconv.Itoa(len(data)))
		c.W.WriteHeader(http.StatusOK)
		_, _ = c.W.Write(data)
		return nil

	case "readme":
		item, err := h.repos.GetReadme(c.Ctx(), in, ref)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)

	case "stats":
		item, err := h.repos.GetStats(c.Ctx(), in, ref)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)

	case "summary":
		item, err := h.repos.GetSummary(c.Ctx(), in, ref)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)

	default:
		return c.Error(http.StatusNotFound, "not_found")
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
	case "branches", "tags", "commits", "tree", "blob", "raw", "blame", "readme", "stats", "summary":
		return last, parts[:len(parts)-1], nil, true
	default:
		return "", nil, nil, false
	}
}

func folderAndName(parts []string) (folderPath, name string, err error) {
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "", "", errors.New("invalid_repository_path")
	}

	name = parts[len(parts)-1]
	if len(parts) > 1 {
		folderPath = strings.Join(parts[:len(parts)-1], "/")
	}

	return folderPath, name, nil
}
