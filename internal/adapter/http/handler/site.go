package handler

import "github.com/magomedcoder/repo/pkg/httpx"

type SiteHandler struct {
	name      string
	publicURL string
}

func NewSiteHandler(name, publicURL string) *SiteHandler {
	return &SiteHandler{
		name:      name,
		publicURL: publicURL,
	}
}

func (h *SiteHandler) Get(c *httpx.Context) error {
	return httpx.OK(c, siteResponse{
		Name:      h.name,
		PublicURL: h.publicURL,
	})
}
