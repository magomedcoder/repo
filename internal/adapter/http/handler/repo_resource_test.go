package handler

import (
	"net/http"
	"testing"

	"github.com/magomedcoder/repo/pkg/httpx"
)

func TestNamedRepoParts(t *testing.T) {
	t.Parallel()

	rootReq := httptestRequest(http.MethodGet, "/repos/solo")
	rootReq.SetPathValue("name", "solo")
	if parts := namedRepoParts(&httpx.Context{R: rootReq}); len(parts) != 1 || parts[0] != "solo" {
		t.Fatalf("root parts=%v", parts)
	}

	emptyReq := httptestRequest(http.MethodGet, "/repos/")
	if parts := namedRepoParts(&httpx.Context{R: emptyReq}); parts != nil {
		t.Fatalf("empty parts=%v", parts)
	}
}

func TestRegisterRepoRoutesNoPanic(t *testing.T) {
	t.Parallel()

	r := httpx.New()
	pub := r.Group("/api")
	authed := r.Group("/api")
	RegisterRepoRoutes(pub, authed, &RepoDispatcher{
		Repos:      &RepositoryHandler{},
		Issues:     &IssueHandler{},
		Pulls:      &PullRequestHandler{},
		Membership: &MembershipHandler{},
	})
}

func TestParseRepoResourceIssuesAndBrowse(t *testing.T) {
	t.Parallel()

	action, parts, extra, ok := ParseRepoResource("demo/issues")
	if !ok || action != "issues" || len(parts) != 1 || parts[0] != "demo" || len(extra) != 0 {
		t.Fatalf("issues: action=%q parts=%v extra=%v ok=%v", action, parts, extra, ok)
	}

	action, parts, extra, ok = ParseRepoResource("demo/issues/3")
	if !ok || action != "issue" || len(parts) != 1 || parts[0] != "demo" || len(extra) != 1 || extra[0] != "3" {
		t.Fatalf("issue: action=%q parts=%v extra=%v ok=%v", action, parts, extra, ok)
	}

	_, nestedParts, _, nestedOK := ParseRepoResource("group/demo/issues/3")
	if nestedOK {
		if _, err := repoNameFromParts(nestedParts); err == nil {
			t.Fatalf("nested path must not yield a repo name: %v", nestedParts)
		}
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

func httptestRequest(method, target string) *http.Request {
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		panic(err)
	}

	return req
}
