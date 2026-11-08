package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"magomedcoder.ru/repo/pkg/httpx"
)

const (
	maxCSPReportBytes = 16 << 10
	maxCSPFieldLen    = 256
)

var cspReportFields = []string{
	"document-uri",
	"violated-directive",
	"effective-directive",
	"blocked-uri",
	"status-code",
	"disposition",
}

func CSPReport(c *httpx.Context) error {
	body, err := io.ReadAll(io.LimitReader(c.R.Body, maxCSPReportBytes+1))
	if err != nil {
		return c.NoContent()
	}

	if len(body) > maxCSPReportBytes {
		return c.Error(http.StatusRequestEntityTooLarge, "payload_too_large")
	}

	if len(body) == 0 {
		return c.NoContent()
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Debug("csp report: invalid json", "err", err)
		return c.NoContent()
	}

	fields := sanitizeCSPReport(payload)
	if len(fields) == 0 {
		return c.NoContent()
	}

	attrs := make([]any, 0, len(fields)*2)
	for _, k := range cspReportFields {
		if v, ok := fields[k]; ok {
			attrs = append(attrs, k, v)
		}
	}

	slog.Info("csp violation", attrs...)

	return c.NoContent()
}

func sanitizeCSPReport(payload map[string]any) map[string]string {
	src := payload
	if inner, ok := payload["csp-report"].(map[string]any); ok {
		src = inner
	}

	out := make(map[string]string, len(cspReportFields))
	for _, k := range cspReportFields {
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

		out[k] = truncateLogField(s, maxCSPFieldLen)
	}

	return out
}

func truncateLogField(s string, n int) string {
	clean := make([]rune, 0, len(s))
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
