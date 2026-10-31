package git

import (
	"path/filepath"
	"testing"
)

func TestSafePathAllowsDotDotInName(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "a..b")
	got, err := SafePath(root, path)
	if err != nil {
		t.Fatalf("SafePath(a..b): %v", err)
	}

	if got != path && filepath.Clean(got) != filepath.Clean(path) {
		t.Fatalf("got %q want %q", got, path)
	}
}

func TestSafePathRejectsParentSegment(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := SafePath(root, filepath.Join(root, "..", "outside"))
	if err == nil {
		t.Fatal("expected error for .. segment")
	}
}
