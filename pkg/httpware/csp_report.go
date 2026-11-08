package httpware

import (
	"encoding/json"
	"unicode/utf8"
)

var CSPReportFields = []string{
	"document-uri",
	"violated-directive",
	"effective-directive",
	"blocked-uri",
	"status-code",
	"disposition",
}

func SanitizeCSPReport(payload map[string]any, maxFieldLen int) map[string]string {
	if maxFieldLen <= 0 {
		maxFieldLen = 256
	}

	src := payload
	if inner, ok := payload["csp-report"].(map[string]any); ok {
		src = inner
	}

	out := make(map[string]string, len(CSPReportFields))
	for _, k := range CSPReportFields {
		v, ok := src[k]
		if !ok {
			continue
		}

		var s string
		switch t := v.(type) {
		case string:
			s = t
		case float64:
			b, _ := json.Marshal(t)
			s = string(b)
		default:
			continue
		}

		out[k] = TruncateLogField(s, maxFieldLen)
	}

	return out
}

func TruncateLogField(s string, n int) string {
	if n <= 0 {
		return ""
	}

	clean := make([]rune, 0, utf8.RuneCountInString(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			r = ' '
		}

		clean = append(clean, r)
		if len(clean) >= n {
			break
		}
	}

	return string(clean)
}
