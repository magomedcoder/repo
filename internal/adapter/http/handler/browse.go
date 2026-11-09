package handler

import (
	"encoding/base64"
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

func (h *RepositoryHandler) serveBrowseGet(c *httpx.Context, in usecase.ResolveRepositoryInput, action string, extra []string) error {
	ref := c.Query("ref")
	filePath := c.Query("path")

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
		items, err := h.repos.ListTree(c.Ctx(), in, ref, filePath)
		if err != nil {
			return mapError(c, err)
		}
		return httpx.OK(c, treeResponse{Ref: ref, Path: filePath, Tree: items})

	case "blob":
		item, err := h.repos.GetBlob(c.Ctx(), in, ref, filePath)
		if err != nil {
			return mapError(c, err)
		}
		return httpx.OK(c, item)

	case "blame":
		item, err := h.repos.GetBlame(c.Ctx(), in, ref, filePath)
		if err != nil {
			return mapError(c, err)
		}

		return httpx.OK(c, item)

	case "raw":
		item, err := h.repos.GetBlob(c.Ctx(), in, ref, filePath)
		if err != nil {
			return mapError(c, err)
		}
		var data []byte
		if item.Size > domain.MaxBlobReadBytes {
			return c.Error(http.StatusRequestEntityTooLarge, "file_too_large")
		}

		if item.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(item.Content)
			if err != nil {
				return c.Error(http.StatusInternalServerError, "internal_error")
			}
			data = decoded
		} else {
			data = []byte(item.Content)
		}

		ctype := rawContentType(filePath, item.IsBinary)
		c.W.Header().Set("Content-Type", ctype)

		name := safeDownloadFilename(path.Base(strings.TrimSpace(filePath)))
		disposition := "attachment"
		if strings.HasPrefix(ctype, "image/") && !strings.EqualFold(ctype, "image/svg+xml") {
			disposition = "inline"
		}

		c.W.Header().Set("Content-Disposition", disposition+`; filename="`+name+`"`)
		c.W.Header().Set("Content-Length", strconv.Itoa(len(data)))
		c.W.WriteHeader(http.StatusOK)
		_, _ = c.W.Write(data)

		if h.access != nil {
			user, _ := middleware.UserFromContext(c.Ctx())
			h.access.Download(c.R, in.Name, ref, filePath, user)
		}

		return nil

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
	case "branches", "tags", "commits", "tree", "blob", "raw", "blame", "summary":
		return last, parts[:len(parts)-1], nil, true
	default:
		return "", nil, nil, false
	}
}

func repoNameFromParts(parts []string) (string, error) {
	if len(parts) != 1 || parts[0] == "" {
		return "", errors.New("invalid_repository_path")
	}

	return parts[0], nil
}

func safeDownloadFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r < 32 || r == 127 || r == '"' || r == '\\' || r == '/' || r == ':':
			return '_'
		default:
			return r
		}
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "download"
	}

	return name
}

func rawContentType(filePath string, isBinary bool) string {
	switch strings.ToLower(path.Ext(filePath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "text/plain; charset=utf-8"
	case ".ico":
		return "image/x-icon"
	case ".bmp":
		return "image/bmp"
	case ".avif":
		return "image/avif"
	}
	if isBinary {
		return "application/octet-stream"
	}
	return "text/plain; charset=utf-8"
}
