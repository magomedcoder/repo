package handler

import (
	"net/http"
	"slices"
	"strings"

	"magomedcoder.ru/repo/pkg/httpx"
)

type FolderDispatcher struct {
	Folders    *FolderHandler
	Membership *MembershipHandler
}

func (d *FolderDispatcher) DispatchGet(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	switch {
	case isFolderMembersPath(rest):
		return d.Membership.ListMembers(c)
	case isFolderTeamsListPath(rest):
		return d.Membership.ListTeams(c)
	case isTeamMembersPath(rest):
		return d.Membership.ListTeamMembers(c)
	case isTeamReposPath(rest):
		return d.Membership.ListTeamRepos(c)
	default:
		return d.Folders.Get(c)
	}
}

func (d *FolderDispatcher) DispatchPost(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	switch {
	case isFolderMembersPath(rest):
		return d.Membership.AddMember(c)
	case isFolderTeamsListPath(rest):
		return d.Membership.CreateTeam(c)
	case isTeamMembersPath(rest):
		return d.Membership.AddTeamMember(c)
	case isTeamReposPath(rest):
		return d.Membership.SetTeamRepo(c)
	default:
		return c.Error(http.StatusNotFound, "not_found")
	}
}

func (d *FolderDispatcher) DispatchPatch(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	switch {
	case isFolderMemberItemPath(rest):
		return d.Membership.UpdateMember(c)
	case isFolderTeamItemPath(rest):
		return d.Membership.UpdateTeam(c)
	default:
		return d.Folders.Rename(c)
	}
}

func (d *FolderDispatcher) DispatchDelete(c *httpx.Context) error {
	rest := strings.Trim(c.Param("path"), "/")
	switch {
	case isFolderMemberItemPath(rest):
		return d.Membership.RemoveMember(c)
	case isFolderTeamItemPath(rest):
		return d.Membership.DeleteTeam(c)
	case isTeamMemberItemPath(rest):
		return d.Membership.RemoveTeamMember(c)
	case isTeamRepoItemPath(rest):
		return d.Membership.RemoveTeamRepo(c)
	default:
		return d.Folders.Delete(c)
	}
}

func isFolderMembersPath(rest string) bool {
	parts := strings.Split(rest, "/")
	return len(parts) >= 2 && parts[len(parts)-1] == "members" && !contains(parts[:len(parts)-1], "teams")
}

func isFolderMemberItemPath(rest string) bool {
	parts := strings.Split(rest, "/")
	return len(parts) >= 3 && parts[len(parts)-2] == "members" && !contains(parts[:len(parts)-2], "teams")
}

func isFolderTeamsListPath(rest string) bool {
	parts := strings.Split(rest, "/")
	return len(parts) >= 2 && parts[len(parts)-1] == "teams"
}

func isFolderTeamItemPath(rest string) bool {
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		return false
	}

	for i, p := range parts {
		if p == "teams" && i+1 == len(parts)-1 {
			return true
		}
	}

	return false
}

func isTeamMembersPath(rest string) bool {
	_, _, rem, ok := splitTeamResource(rest, "members")
	return ok && rem == ""
}

func isTeamMemberItemPath(rest string) bool {
	_, _, rem, ok := splitTeamResource(rest, "members")
	return ok && rem != "" && !strings.Contains(rem, "/")
}

func isTeamReposPath(rest string) bool {
	_, _, rem, ok := splitTeamResource(rest, "repos")
	return ok && rem == ""
}

func isTeamRepoItemPath(rest string) bool {
	_, _, rem, ok := splitTeamResource(rest, "repos")
	return ok && rem != "" && !strings.Contains(rem, "/")
}

func contains(parts []string, s string) bool {
	return slices.Contains(parts, s)
}
