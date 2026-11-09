package middleware

import (
	"net/http"

	"github.com/magomedcoder/repo/pkg/httpware"
)

func Logger(next http.Handler) http.Handler {
	return httpware.Logger(next)
}
