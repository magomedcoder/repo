package handler

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
)

type registrationResponse struct {
	AllowRegistration bool `json:"allow_registration"`
}

type userJSON struct {
	ID        uint        `json:"id"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	Role      domain.Role `json:"role"`
	HasAvatar bool        `json:"has_avatar"`
	CreatedAt time.Time   `json:"created_at"`
}

type siteResponse struct {
	Name string `json:"name"`
}

type stepUpRequest struct {
	CurrentPassword string `json:"current_password"`
}

type passwordChangeRequest struct {
	Current string `json:"current"`
	Next    string `json:"next"`
}

type reposResponse struct {
	Repos []usecase.RepositoryItem `json:"repos"`
}

type tokensResponse struct {
	Tokens []usecase.TokenItem `json:"tokens"`
}

type activityResponse struct {
	Events []usecase.ActivityItem `json:"events"`
}

type commentWriteRequest struct {
	Body string `json:"body"`
}

type issuesResponse struct {
	Issues []usecase.IssueItem `json:"issues"`
}

type labelsResponse struct {
	Labels []usecase.LabelItem `json:"labels"`
}

type pullsResponse struct {
	Pulls []usecase.PullItem `json:"pulls"`
}

type commitsResponse struct {
	Commits []domain.CommitInfo `json:"commits"`
}

type branchesResponse struct {
	Branches []domain.RefInfo `json:"branches"`
}

type tagsResponse struct {
	Tags []domain.RefInfo `json:"tags"`
}

type treeResponse struct {
	Ref  string             `json:"ref"`
	Path string             `json:"path"`
	Tree []domain.TreeEntry `json:"tree"`
}

func userResponse(user *domain.User) userJSON {
	role := user.Role
	if !role.Valid() {
		role = domain.RoleUser
	}

	return userJSON{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      role,
		HasAvatar: user.AvatarPath != "",
		CreatedAt: user.CreatedAt,
	}
}
