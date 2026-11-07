package handler

import (
	"net/http"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

func (h *RepositoryHandler) Search(c *httpx.Context) error {
	q := strings.TrimSpace(c.Query("q"))
	typ := strings.ToLower(strings.TrimSpace(c.Query("type")))
	if typ == "" {
		typ = "repositories"
	}

	var viewerID uint
	if user, ok := middleware.UserFromContext(c.Ctx()); ok {
		viewerID = user.ID
	}

	switch typ {
	case "repositories", "repos":
		offset, limit := usecase.ParseOffsetLimit(c.Query("offset"), c.Query("limit"))
		offset, limit = usecase.NormalizePagination(offset, limit)
		items, err := h.repos.SearchAccessibleRepos(c.Ctx(), q, viewerID, limit, offset)
		if err != nil {
			return mapError(c, err)
		}

		if items == nil {
			items = []usecase.RepositoryItem{}
		}

		return httpx.OK(c, map[string]any{
			"type":  "repositories",
			"repos": items,
		})

	case "code":
		items, err := h.repos.SearchCode(c.Ctx(), q, viewerID)
		if err != nil {
			return mapError(c, err)
		}

		if items == nil {
			items = []usecase.CodeSearchResult{}
		}

		return httpx.OK(c, map[string]any{
			"type":  "code",
			"items": items,
		})

	default:
		return c.Error(http.StatusBadRequest, "invalid_search_type")
	}
}
