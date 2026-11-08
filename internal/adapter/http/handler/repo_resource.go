package handler

import "strings"

func ParseRepoResource(rest string) (action string, repoParts, extra []string, ok bool) {
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 {
		return "", nil, nil, false
	}

	if a, rp, ex, ok := splitPullPath(parts); ok {
		return a, rp, ex, true
	}

	if a, rp, ex, ok := splitIssuePath(parts); ok {
		return a, rp, ex, true
	}

	if a, rp, ex, ok := splitBrowsePath(parts); ok {
		return a, rp, ex, true
	}

	if parts[len(parts)-1] == "collaborators" {
		return "collaborators", parts[:len(parts)-1], nil, true
	}

	if len(parts) >= 2 && parts[len(parts)-2] == "collaborators" {
		return "collaborator", parts[:len(parts)-2], []string{parts[len(parts)-1]}, true
	}

	if parts[len(parts)-1] == "teams" {
		return "teams", parts[:len(parts)-1], nil, true
	}

	if len(parts) >= 2 && parts[len(parts)-2] == "teams" {
		return "team", parts[:len(parts)-2], []string{parts[len(parts)-1]}, true
	}

	return "", nil, nil, false
}

func isPullAction(action string) bool {
	switch action {
	case "pulls", "pull", "pull-diff", "pull-merge", "pull-comments", "pull-comment":
		return true
	default:
		return false
	}
}

func isIssueAction(action string) bool {
	switch action {
	case "issues", "issue", "labels", "label", "issue-comments", "issue-comment":
		return true
	default:
		return false
	}
}
