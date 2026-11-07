package handler

import (
	"net/http"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type RepoDispatcher struct {
	Repos      *RepositoryHandler
	Issues     *IssueHandler
	Pulls      *PullRequestHandler
	Membership *MembershipHandler
	Teams      *TeamHandler
}

func (d *RepoDispatcher) DispatchGet(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	if rest == "" {
		return c.Error(http.StatusBadRequest, "invalid_repository_path")
	}

	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		return d.Repos.Get(c)
	}

	in, err := repoInputFromParts(c, repoParts)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	switch {
	case action == "collaborators":
		if _, ok := middleware.UserFromContext(c.Ctx()); !ok {
			return c.Error(http.StatusUnauthorized, "unauthorized")
		}
		return d.Membership.ListCollaborators(c, in)
	case action == "teams":
		if d.Teams == nil {
			return c.Error(http.StatusNotFound, "not_found")
		}
		return d.Teams.ListRepoTeams(c, in)
	case isPullAction(action):
		return d.Pulls.get(c, in, action, extra)
	case isIssueAction(action):
		return d.Issues.get(c, in, action, extra)
	default:
		return d.Repos.serveBrowseGet(c, in, action, extra)
	}
}

func (d *RepoDispatcher) DispatchPatch(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		return d.Repos.Update(c)
	}

	in, err := repoInputFromParts(c, repoParts)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	switch {
	case isPullAction(action):
		return d.Pulls.patch(c, in, action, extra)
	case isIssueAction(action):
		return d.Issues.patch(c, in, action, extra)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (d *RepoDispatcher) DispatchDelete(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		return d.Repos.Delete(c)
	}

	in, err := repoInputFromParts(c, repoParts)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	switch {
	case action == "collaborator" && len(extra) == 1:
		return d.Membership.RemoveCollaborator(c, in, extra[0])
	case action == "team" && len(extra) == 1:
		if d.Teams == nil {
			return c.Error(http.StatusNotFound, "not_found")
		}

		return d.Teams.RevokeRepoTeam(c, in, extra[0])
	case isPullAction(action):
		return d.Pulls.delete(c, in, action, extra)
	case isIssueAction(action):
		return d.Issues.delete(c, in, action, extra)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (d *RepoDispatcher) DispatchPost(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		return c.Error(http.StatusNotFound, "not_found")
	}

	in, err := repoInputFromParts(c, repoParts)
	if err != nil {
		return c.Error(http.StatusBadRequest, err.Error())
	}

	switch {
	case action == "collaborators":
		return d.Membership.AddCollaborator(c, in)
	case action == "teams":
		if d.Teams == nil {
			return c.Error(http.StatusNotFound, "not_found")
		}
		return d.Teams.GrantRepoTeam(c, in)
	case isPullAction(action):
		return d.Pulls.post(c, in, action, extra)
	case isIssueAction(action):
		return d.Issues.post(c, in, action, extra)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func repoInputFromParts(c *httpx.Context, repoParts []string) (usecase.ResolveRepositoryInput, error) {
	name, err := repoNameFromParts(repoParts)
	if err != nil {
		return usecase.ResolveRepositoryInput{}, err
	}

	var viewerID uint
	if user, loggedIn := middleware.UserFromContext(c.Ctx()); loggedIn {
		viewerID = user.ID
	}

	return usecase.ResolveRepositoryInput{
		Name:     name,
		ViewerID: viewerID,
	}, nil
}
