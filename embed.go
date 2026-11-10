package repo

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/magomedcoder/repo/pkg/spa"
)

//go:embed all:web/dist
var webDist embed.FS

func WebFS() (fs.FS, error) {
	return fs.Sub(webDist, "web/dist")
}

func WebHandler() (http.Handler, error) {
	sub, err := WebFS()
	if err != nil {
		return nil, err
	}

	return spa.FileServer(sub, spa.Options{}), nil
}
