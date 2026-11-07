package domain

import "strings"

var reservedSlugs = map[string]struct{}{
	"api":      {},
	"cp":       {},
	"data":     {},
	"login":    {},
	"register": {},
	"settings": {},
	"search":   {},
	"assets":   {},
	"static":   {},
	"users":    {},
	"me":       {},
	"issues":   {},
	"pulls":    {},
	"commits":  {},
	"commit":   {},
	"tree":     {},
	"blob":     {},
	"refs":     {},
	"raw":      {},
}

func IsReservedSlug(slug string) bool {
	_, ok := reservedSlugs[strings.ToLower(strings.TrimSpace(slug))]
	return ok
}
