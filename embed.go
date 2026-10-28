package repo

import (
	"embed"
	"net/http"

	"repo.magomedcoder.ru/repo/pkg/spa"
)

//go:embed all:web/dist
var webDist embed.FS

func WebHandler() (http.Handler, error) {
	return spa.Sub(webDist, "web/dist", spa.Options{})
}
