package git

import (
	"strings"
	"testing"
)

func TestSanitizeGrepContent(t *testing.T) {
	t.Parallel()

	got := sanitizeGrepContent("a\tb")
	if got != "a b" {
		t.Fatalf("tabs: %q", got)
	}

	long := strings.Repeat("x", maxGrepLineLen+10)
	got = sanitizeGrepContent(long)
	if !strings.HasSuffix(got, "...") || len(got) <= maxGrepLineLen {
		t.Fatalf("truncate: len=%d got=%q", len(got), got)
	}
}
