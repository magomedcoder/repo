package domain

import "strings"

var reservedSlugs = map[string]struct{}{
	"api":              {},
	"cp":               {},
	"data":             {},
	"login":            {},
	"register":         {},
	"settings":         {},
	"search":           {},
	"assets":           {},
	"static":           {},
	"users":            {},
	"me":               {},
	"issues":           {},
	"pulls":            {},
	"commits":          {},
	"commit":           {},
	"tree":             {},
	"blob":             {},
	"refs":             {},
	"raw":              {},
	"robots.txt":       {},
	"sitemap.xml":      {},
	"og.png":           {},
	"site.webmanifest": {},
}

func IsReservedSlug(slug string) bool {
	_, ok := reservedSlugs[strings.ToLower(strings.TrimSpace(slug))]
	return ok
}
