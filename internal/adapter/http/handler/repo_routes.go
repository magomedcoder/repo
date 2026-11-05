package handler

import (
	"net/http"
	"strings"

	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

func RegisterRepoRoutes(pub, authed *httpx.Router, d *RepoDispatcher) {
	mountRepoCRUD(pub, authed, d, "/repos/{name}")
	mountRepoCollectionRoutes(pub, authed, d, "/repos/{name}")

	mountRepoCRUD(pub, authed, d, "/repos/{folder}/{name}")
	mountRepoCollectionRoutes(pub, authed, d, "/repos/{folder}/{name}")
	mountRepoDetailRoutes(pub, authed, d, "/repos/{folder}/{name}")

	pub.GET("/repos/{path...}", d.DispatchGet)
	authed.PATCH("/repos/{path...}", d.DispatchPatch)
	authed.DELETE("/repos/{path...}", d.DispatchDelete)
	authed.POST("/repos/{path...}", d.DispatchPost)
}

func mountRepoCRUD(pub, authed *httpx.Router, d *RepoDispatcher, base string) {
	pub.GET(base, d.Repos.Get)
	authed.PATCH(base, d.Repos.Update)
	authed.DELETE(base, d.Repos.Delete)
}

func mountRepoCollectionRoutes(pub, authed *httpx.Router, d *RepoDispatcher, base string) {
	pub.GET(base+"/issues", d.explicitGet("issues"))
	pub.GET(base+"/labels", d.explicitGet("labels"))
	authed.POST(base+"/issues", d.explicitPost("issues"))
	authed.POST(base+"/labels", d.explicitPost("labels"))

	pub.GET(base+"/pulls", d.explicitGet("pulls"))
	authed.POST(base+"/pulls", d.explicitPost("pulls"))

	for _, action := range []string{"branches", "tags", "commits", "tree", "blob", "raw", "blame", "readme", "stats", "summary"} {
		pub.GET(base+"/"+action, d.explicitGet(action))
	}

	authed.POST(base+"/move", d.explicitPost("move"))

	authed.PUT(base+"/mirror", d.Repos.ConfigureMirror)
	authed.POST(base+"/mirror/sync", d.Repos.SyncMirror)

	authed.GET(base+"/collaborators", d.explicitGet("collaborators"))
	authed.POST(base+"/collaborators", d.explicitPost("collaborators"))
}

func mountRepoDetailRoutes(pub, authed *httpx.Router, d *RepoDispatcher, base string) {
	pub.GET(base+"/issues/{number}", d.explicitGet("issue"))
	authed.POST(base+"/issues/{number}/comments", d.explicitPost("issue-comments"))
	authed.PATCH(base+"/issues/{number}", d.explicitPatch("issue"))
	authed.PATCH(base+"/issues/{number}/comments/{commentID}", d.explicitPatch("issue-comment"))
	authed.DELETE(base+"/issues/{number}", d.explicitDelete("issue"))
	authed.DELETE(base+"/issues/{number}/comments/{commentID}", d.explicitDelete("issue-comment"))
	authed.PATCH(base+"/labels/{labelID}", d.explicitPatch("label"))
	authed.DELETE(base+"/labels/{labelID}", d.explicitDelete("label"))

	pub.GET(base+"/pulls/{number}", d.explicitGet("pull"))
	pub.GET(base+"/pulls/{number}/diff", d.explicitGet("pull-diff"))
	pub.GET(base+"/pulls/{number}/commits", d.explicitGet("pull-commits"))
	authed.POST(base+"/pulls/{number}/comments", d.explicitPost("pull-comments"))
	authed.POST(base+"/pulls/{number}/merge", d.explicitPost("pull-merge"))
	authed.PATCH(base+"/pulls/{number}", d.explicitPatch("pull"))
	authed.PATCH(base+"/pulls/{number}/comments/{commentID}", d.explicitPatch("pull-comment"))
	authed.DELETE(base+"/pulls/{number}", d.explicitDelete("pull"))
	authed.DELETE(base+"/pulls/{number}/comments/{commentID}", d.explicitDelete("pull-comment"))

	pub.GET(base+"/commits/{sha}", d.explicitGet("commits"))
	pub.GET(base+"/commits/{sha}/diff", d.explicitGet("commit-diff"))

	authed.DELETE(base+"/collaborators/{username}", d.explicitDelete("collaborator"))
}

func (d *RepoDispatcher) explicitGet(action string) httpx.Handler {
	return func(c *httpx.Context) error {
		in, err := resolveNamedRepo(c)
		if err != nil {
			return c.Error(http.StatusBadRequest, err.Error())
		}
		extra := explicitExtra(c, action)
		switch {
		case action == "collaborators":
			return d.Membership.ListCollaborators(c, in)
		case action == "commit-diff":
			return d.Repos.serveBrowseGet(c, in, "commits", []string{c.Param("sha"), "diff"})
		case isPullAction(action):
			return d.Pulls.get(c, in, action, extra)
		case isIssueAction(action):
			return d.Issues.get(c, in, action, extra)
		default:
			return d.Repos.serveBrowseGet(c, in, action, extra)
		}
	}
}

func (d *RepoDispatcher) explicitPost(action string) httpx.Handler {
	return func(c *httpx.Context) error {
		if action == "move" {
			return d.Repos.MoveWithParts(c, namedRepoParts(c))
		}
		in, err := resolveNamedRepo(c)
		if err != nil {
			return c.Error(http.StatusBadRequest, err.Error())
		}
		extra := explicitExtra(c, action)
		switch {
		case action == "collaborators":
			return d.Membership.AddCollaborator(c, in)
		case isPullAction(action):
			return d.Pulls.post(c, in, action, extra)
		case isIssueAction(action):
			return d.Issues.post(c, in, action, extra)
		default:
			return c.Error(http.StatusNotFound, "not_found")
		}
	}
}

func (d *RepoDispatcher) explicitPatch(action string) httpx.Handler {
	return func(c *httpx.Context) error {
		in, err := resolveNamedRepo(c)
		if err != nil {
			return c.Error(http.StatusBadRequest, err.Error())
		}
		extra := explicitExtra(c, action)
		switch {
		case isPullAction(action):
			return d.Pulls.patch(c, in, action, extra)
		case isIssueAction(action):
			return d.Issues.patch(c, in, action, extra)
		default:
			return c.Error(http.StatusNotFound, "not_found")
		}
	}
}

func (d *RepoDispatcher) explicitDelete(action string) httpx.Handler {
	return func(c *httpx.Context) error {
		in, err := resolveNamedRepo(c)
		if err != nil {
			return c.Error(http.StatusBadRequest, err.Error())
		}

		extra := explicitExtra(c, action)
		switch {
		case action == "collaborator":
			return d.Membership.RemoveCollaborator(c, in, c.Param("username"))
		case isPullAction(action):
			return d.Pulls.delete(c, in, action, extra)
		case isIssueAction(action):
			return d.Issues.delete(c, in, action, extra)
		default:
			return c.Error(http.StatusNotFound, "not_found")
		}
	}
}

func resolveNamedRepo(c *httpx.Context) (usecase.ResolveRepositoryInput, error) {
	return repoInputFromParts(c, namedRepoParts(c))
}

func namedRepoParts(c *httpx.Context) []string {
	name := strings.Trim(c.Param("name"), "/")
	folder := strings.Trim(c.Param("folder"), "/")
	if folder != "" {
		return []string{folder, name}
	}

	if name == "" {
		return nil
	}

	return []string{name}
}

func explicitExtra(c *httpx.Context, action string) []string {
	switch action {
	case "issue", "pull", "pull-diff", "pull-commits", "pull-merge", "pull-comments", "issue-comments", "label":
		if n := c.Param("number"); n != "" {
			return []string{n}
		}
		if id := c.Param("labelID"); id != "" {
			return []string{id}
		}
	case "issue-comment", "pull-comment":
		return []string{c.Param("number"), c.Param("commentID")}
	case "commits":
		if sha := c.Param("sha"); sha != "" {
			return []string{sha}
		}
	}

	return nil
}
