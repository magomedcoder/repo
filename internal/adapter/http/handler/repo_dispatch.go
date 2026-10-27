package handler

import (
	"net/http"
	"strings"

	"git.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"git.magomedcoder.ru/repo/internal/usecase"
)

type RepoDispatcher struct {
	Repos  *RepositoryHandler
	Issues *IssueHandler
	Pulls  *PullRequestHandler
}

func (d *RepoDispatcher) DispatchGet(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(r.PathValue("path"), "/")
	if rest == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_repository_path")
		return
	}

	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		d.Repos.Get(w, r)
		return
	}

	in, err := repoInputFromParts(r, repoParts)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	switch {
	case isPullAction(action):
		d.Pulls.get(w, r, in, action, extra)
	case isIssueAction(action):
		d.Issues.get(w, r, in, action, extra)
	default:
		d.Repos.serveBrowseGet(w, r, in, action, extra)
	}
}

func (d *RepoDispatcher) DispatchPatch(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(r.PathValue("path"), "/")
	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		d.Repos.Update(w, r)
		return
	}

	in, err := repoInputFromParts(r, repoParts)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	switch {
	case isPullAction(action):
		d.Pulls.patch(w, r, in, action, extra)
	case isIssueAction(action):
		d.Issues.patch(w, r, in, action, extra)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (d *RepoDispatcher) DispatchDelete(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(r.PathValue("path"), "/")
	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		d.Repos.Delete(w, r)
		return
	}

	in, err := repoInputFromParts(r, repoParts)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	switch {
	case isPullAction(action):
		d.Pulls.delete(w, r, in, action, extra)
	case isIssueAction(action):
		d.Issues.delete(w, r, in, action, extra)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func (d *RepoDispatcher) DispatchPost(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(r.PathValue("path"), "/")
	action, repoParts, extra, ok := ParseRepoResource(rest)
	if !ok {
		writeError(w, r, http.StatusNotFound, "not_found")
		return
	}

	if action == "move" {
		d.Repos.MoveWithParts(w, r, repoParts)
		return
	}

	in, err := repoInputFromParts(r, repoParts)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	switch {
	case isPullAction(action):
		d.Pulls.post(w, r, in, action, extra)
	case isIssueAction(action):
		d.Issues.post(w, r, in, action, extra)
	default:
		writeError(w, r, http.StatusNotFound, "not_found")
	}
}

func repoInputFromParts(r *http.Request, repoParts []string) (usecase.ResolveRepositoryInput, error) {
	folderPath, name, err := folderAndName(repoParts)
	if err != nil {
		return usecase.ResolveRepositoryInput{}, err
	}

	var viewerID uint
	if user, loggedIn := middleware.UserFromContext(r.Context()); loggedIn {
		viewerID = user.ID
	}

	return usecase.ResolveRepositoryInput{
		FolderPath: folderPath,
		Name:       name,
		ViewerID:   viewerID,
	}, nil
}
