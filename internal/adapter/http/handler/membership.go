package handler

import (
	"net/http"
	"strings"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type MembershipHandler struct {
	access *usecase.AccessUseCase
}

func NewMembershipHandler(access *usecase.AccessUseCase) *MembershipHandler {
	return &MembershipHandler{access: access}
}

type memberRequest struct {
	Username string          `json:"username"`
	Role     string          `json:"role"`
	Caps     map[string]bool `json:"caps"`
}

type teamRequest struct {
	Name string          `json:"name"`
	Caps map[string]bool `json:"caps"`
}

type teamMemberRequest struct {
	Username string `json:"username"`
}

type teamRepoRequest struct {
	Repo string `json:"repo"`
	Role string `json:"role"`
}

type collaboratorRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (h *MembershipHandler) ListMembers(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, rest := splitFolderSuffix(c.Param("path"), "members")
	if rest != "" || path == "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	out, err := h.access.ListMembers(c.Ctx(), user.ID, path)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"members": out})
}

func (h *MembershipHandler) AddMember(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, rest := splitFolderSuffix(c.Param("path"), "members")
	if rest != "" || path == "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	req, err := httpx.BindJSON[memberRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.access.AddMember(c.Ctx(), usecase.UpsertMemberInput{
		ViewerID:   user.ID,
		FolderPath: path,
		Username:   req.Username,
		Role:       req.Role,
		Caps:       req.Caps,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *MembershipHandler) UpdateMember(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, username := splitFolderSuffix(c.Param("path"), "members")
	if path == "" || username == "" || strings.Contains(username, "/") {
		return c.Error(http.StatusNotFound, "not_found")
	}

	req, err := httpx.BindJSON[memberRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	if req.Username == "" {
		req.Username = username
	}

	out, err := h.access.UpdateMember(c.Ctx(), usecase.UpsertMemberInput{
		ViewerID:   user.ID,
		FolderPath: path,
		Username:   req.Username,
		Role:       req.Role,
		Caps:       req.Caps,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *MembershipHandler) RemoveMember(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, username := splitFolderSuffix(c.Param("path"), "members")
	if path == "" || username == "" || strings.Contains(username, "/") {
		return c.Error(http.StatusNotFound, "not_found")
	}

	if err := h.access.RemoveMember(c.Ctx(), user.ID, path, username); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *MembershipHandler) ListTeams(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, rest := splitFolderSuffix(c.Param("path"), "teams")
	if rest != "" || path == "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	out, err := h.access.ListTeams(c.Ctx(), user.ID, path)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"teams": out})
}

func (h *MembershipHandler) CreateTeam(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, rest := splitFolderSuffix(c.Param("path"), "teams")
	if rest != "" || path == "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	req, err := httpx.BindJSON[teamRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.access.CreateTeam(c.Ctx(), usecase.UpsertTeamInput{
		ViewerID:   user.ID,
		FolderPath: path,
		Name:       req.Name,
		Caps:       req.Caps,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *MembershipHandler) UpdateTeam(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug := splitFolderSuffix(c.Param("path"), "teams")
	if path == "" || slug == "" || strings.Contains(slug, "/") {
		return c.Error(http.StatusNotFound, "not_found")
	}

	req, err := httpx.BindJSON[teamRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.access.UpdateTeam(c.Ctx(), path, slug, usecase.UpsertTeamInput{
		ViewerID:   user.ID,
		FolderPath: path,
		Name:       req.Name,
		Caps:       req.Caps,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *MembershipHandler) DeleteTeam(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug := splitFolderSuffix(c.Param("path"), "teams")
	if path == "" || slug == "" || strings.Contains(slug, "/") {
		return c.Error(http.StatusNotFound, "not_found")
	}

	if err := h.access.DeleteTeam(c.Ctx(), user.ID, path, slug); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *MembershipHandler) ListTeamMembers(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug, rest, ok := splitTeamResource(c.Param("path"), "members")
	if !ok || rest != "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	out, err := h.access.ListTeamMembers(c.Ctx(), user.ID, path, slug)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"members": out})
}

func (h *MembershipHandler) AddTeamMember(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug, rest, ok := splitTeamResource(c.Param("path"), "members")
	if !ok || rest != "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	req, err := httpx.BindJSON[teamMemberRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.access.AddTeamMember(c.Ctx(), user.ID, path, slug, req.Username)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *MembershipHandler) RemoveTeamMember(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug, username, ok := splitTeamResource(c.Param("path"), "members")
	if !ok || username == "" || strings.Contains(username, "/") {
		return c.Error(http.StatusNotFound, "not_found")
	}

	if err := h.access.RemoveTeamMember(c.Ctx(), user.ID, path, slug, username); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *MembershipHandler) ListTeamRepos(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug, rest, ok := splitTeamResource(c.Param("path"), "repos")
	if !ok || rest != "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	out, err := h.access.ListTeamRepos(c.Ctx(), user.ID, path, slug)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"repos": out})
}

func (h *MembershipHandler) SetTeamRepo(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug, rest, ok := splitTeamResource(c.Param("path"), "repos")
	if !ok || rest != "" {
		return c.Error(http.StatusNotFound, "not_found")
	}

	req, err := httpx.BindJSON[teamRepoRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.access.SetTeamRepo(c.Ctx(), usecase.SetTeamRepoInput{
		ViewerID:   user.ID,
		FolderPath: path,
		TeamSlug:   slug,
		RepoName:   req.Repo,
		Role:       req.Role,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *MembershipHandler) RemoveTeamRepo(c *httpx.Context) error {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	path, slug, repoName, ok := splitTeamResource(c.Param("path"), "repos")
	if !ok || repoName == "" || strings.Contains(repoName, "/") {
		return c.Error(http.StatusNotFound, "not_found")
	}

	if err := h.access.RemoveTeamRepo(c.Ctx(), user.ID, path, slug, repoName); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *MembershipHandler) ListCollaborators(c *httpx.Context, in usecase.ResolveRepositoryInput) error {
	out, err := h.access.ListCollaborators(c.Ctx(), in)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"collaborators": out})
}

func (h *MembershipHandler) AddCollaborator(c *httpx.Context, in usecase.ResolveRepositoryInput) error {
	req, err := httpx.BindJSON[collaboratorRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.access.AddCollaborator(c.Ctx(), usecase.UpsertCollaboratorInput{
		ResolveRepositoryInput: in,
		Username:               req.Username,
		Role:                   req.Role,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *MembershipHandler) RemoveCollaborator(c *httpx.Context, in usecase.ResolveRepositoryInput, username string) error {
	if err := h.access.RemoveCollaborator(c.Ctx(), in, username); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func splitFolderSuffix(raw, marker string) (folderPath, rest string) {
	raw = strings.Trim(raw, "/")
	parts := strings.Split(raw, "/")
	if len(parts) >= 2 && parts[1] == marker {
		return parts[0], strings.Join(parts[2:], "/")
	}
	return "", ""
}

func splitTeamResource(raw, resource string) (folderPath, teamSlug, rest string, ok bool) {
	raw = strings.Trim(raw, "/")
	parts := strings.Split(raw, "/")
	if len(parts) >= 4 && parts[1] == "teams" && parts[3] == resource {
		folderPath = parts[0]
		teamSlug = parts[2]
		rest = strings.Join(parts[4:], "/")
		return folderPath, teamSlug, rest, folderPath != "" && teamSlug != ""
	}
	return "", "", "", false
}
