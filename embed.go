package repo

import (
	"embed"
	"net/http"

	"github.com/magomedcoder/repo/pkg/spa"
)

//go:embed all:web/dist
var webDist embed.FS

func WebHandler() (http.Handler, error) {
	return spa.Sub(webDist, "web/dist", spa.Options{})
}
