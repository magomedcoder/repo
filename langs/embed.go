package langs

import "embed"

//go:embed en/api.json en/web.json ru/api.json ru/web.json
var FS embed.FS
