package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/pkg/httpware"
	"github.com/magomedcoder/repo/pkg/httpx"
)

const (
	maxCSPReportBytes = 16 << 10
	maxCSPFieldLen    = 256
)

var cspReportLimiter = httpware.NewRateLimiter(30, 10)

func CSPReport(c *httpx.Context) error {
	if !cspReportLimiter.Allow(middleware.ClientIPKey(c.R)) {
		httpware.WriteRateLimited(c.W)
		return nil
	}

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
		slog.Debug("csp", "err", err)
		return c.NoContent()
	}

	fields := httpware.SanitizeCSPReport(payload, maxCSPFieldLen)
	if len(fields) == 0 {
		return c.NoContent()
	}

	attrs := make([]any, 0, len(fields)*2)
	for _, k := range httpware.CSPReportFields {
		if v, ok := fields[k]; ok {
			attrs = append(attrs, k, v)
		}
	}

	slog.Info("csp", attrs...)

	return c.NoContent()
}
