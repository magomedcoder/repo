package middleware

import (
	"net/http"

	"magomedcoder.ru/repo/pkg/httpware"
)

func Logger(next http.Handler) http.Handler {
	return httpware.Logger(next)
}
