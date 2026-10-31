package handler

import "magomedcoder.ru/repo/pkg/httpx"

type SiteHandler struct {
	name string
}

func NewSiteHandler(name string) *SiteHandler {
	return &SiteHandler{name: name}
}

func (h *SiteHandler) Get(c *httpx.Context) error {
	return httpx.OK(c, siteResponse{Name: h.name})
}
