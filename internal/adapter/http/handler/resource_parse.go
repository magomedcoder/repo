package handler

import (
	"net/http"
	"strconv"

	"magomedcoder.ru/repo/pkg/httpx"
)

func parsePositive(raw string) (int, bool) {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, false
	}

	return n, true
}

func requirePositive(c *httpx.Context, raw, notFoundCode string) (int, error) {
	n, ok := parsePositive(raw)
	if !ok {
		return 0, c.Error(http.StatusNotFound, notFoundCode)
	}

	return n, nil
}

func requireExtraPositive(c *httpx.Context, extra []string, notFoundCode string) (int, error) {
	if len(extra) < 1 {
		return 0, c.Error(http.StatusNotFound, notFoundCode)
	}

	return requirePositive(c, extra[0], notFoundCode)
}

func requireCommentIDs(c *httpx.Context, extra []string) (number, commentID int, err error) {
	if len(extra) < 2 {
		return 0, 0, c.Error(http.StatusNotFound, "comment_not_found")
	}

	number, ok := parsePositive(extra[0])
	commentID, okID := parsePositive(extra[1])
	if !ok || !okID {
		return 0, 0, c.Error(http.StatusNotFound, "comment_not_found")
	}

	return number, commentID, nil
}

func bindJSON[T any](c *httpx.Context) (T, error) {
	req, err := httpx.BindJSON[T](c)
	if err != nil {
		var zero T

		return zero, c.Error(http.StatusBadRequest, "invalid_json")
	}

	return req, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

func splitNamedResource(parts []string, collection, singular string) (action string, repoParts, extra []string, ok bool) {
	n := len(parts)
	if n < 2 {
		return "", nil, nil, false
	}

	if n >= 4 && parts[n-2] == "comments" && parts[n-4] == collection {
		return singular + "-comment", parts[:n-4], []string{parts[n-3], parts[n-1]}, true
	}

	if n >= 3 && parts[n-1] == "comments" && parts[n-3] == collection {
		return singular + "-comments", parts[:n-3], []string{parts[n-2]}, true
	}

	if parts[n-1] == collection {
		return collection, parts[:n-1], nil, true
	}

	if parts[n-2] == collection {
		return singular, parts[:n-2], []string{parts[n-1]}, true
	}

	return "", nil, nil, false
}
