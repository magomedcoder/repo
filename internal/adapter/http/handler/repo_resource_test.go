package handler

import "testing"

func TestParseRepoResourceIssuesAndBrowse(t *testing.T) {
	t.Parallel()

	action, parts, extra, ok := ParseRepoResource("demo/issues")
	if !ok || action != "issues" || len(parts) != 1 || parts[0] != "demo" || len(extra) != 0 {
		t.Fatalf("issues: action=%q parts=%v extra=%v ok=%v", action, parts, extra, ok)
	}

	action, parts, extra, ok = ParseRepoResource("folder/demo/issues/3")
	if !ok || action != "issue" || len(parts) != 2 || parts[0] != "folder" || parts[1] != "demo" || len(extra) != 1 || extra[0] != "3" {
		t.Fatalf("issue: action=%q parts=%v extra=%v ok=%v", action, parts, extra, ok)
	}

	action, parts, extra, ok = ParseRepoResource("demo/branches")
	if !ok || action != "branches" || len(parts) != 1 || parts[0] != "demo" {
		t.Fatalf("branches: action=%q parts=%v extra=%v ok=%v", action, parts, extra, ok)
	}

	_, _, _, ok = ParseRepoResource("demo")
	if ok {
		t.Fatal("bare repo path should not parse as resource action")
	}
}
