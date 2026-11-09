package middleware

import (
	"net/http"

	"github.com/magomedcoder/repo/pkg/httpware"
)

func MaxBody(n int64) func(http.Handler) http.Handler {
	return httpware.MaxBody(n)
}
