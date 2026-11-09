package handler

import (
	"errors"
	"net/http"

	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

func mapError(c *httpx.Context, err error) error {
	if err == nil {
		return nil
	}

	if appErr, ok := errors.AsType[*usecase.Error](err); ok {
		return c.Error(statusFor(appErr.Kind), appErr.Code)
	}

	return c.Error(http.StatusInternalServerError, "internal_error")
}

func statusFor(kind usecase.Kind) int {
	switch kind {
	case usecase.KindValidation:
		return http.StatusBadRequest
	case usecase.KindUnauthorized:
		return http.StatusUnauthorized
	case usecase.KindForbidden:
		return http.StatusForbidden
	case usecase.KindNotFound:
		return http.StatusNotFound
	case usecase.KindConflict:
		return http.StatusConflict
	case usecase.KindRateLimited:
		return http.StatusTooManyRequests
	case usecase.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
