package handler

import (
	"net/http"

	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

type TeamHandler struct {
	teams *usecase.TeamUseCase
}

func NewTeamHandler(teams *usecase.TeamUseCase) *TeamHandler {
	return &TeamHandler{teams: teams}
}

type teamRequest struct {
	Name string `json:"name"`
}

type teamMemberRequest struct {
	Username string `json:"username"`
}

type teamRepoRequest struct {
	Repo string `json:"repo"`
	Role string `json:"role"`
}

type repoTeamRequest struct {
	Team string `json:"team"`
	Role string `json:"role"`
}

func (h *TeamHandler) viewer(c *httpx.Context) (uint, bool) {
	user, ok := middleware.UserFromContext(c.Ctx())
	if !ok {
		return 0, false
	}

	return user.ID, true
}

func (h *TeamHandler) List(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.teams.List(c.Ctx(), uid)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"teams": out})
}

func (h *TeamHandler) Create(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[teamRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.teams.Create(c.Ctx(), uid, req.Name)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *TeamHandler) Get(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.teams.Get(c.Ctx(), uid, c.Param("slug"))
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *TeamHandler) Update(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[teamRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.teams.Update(c.Ctx(), uid, c.Param("slug"), req.Name)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *TeamHandler) Delete(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	if err := h.teams.Delete(c.Ctx(), uid, c.Param("slug")); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *TeamHandler) ListMembers(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.teams.ListMembers(c.Ctx(), uid, c.Param("slug"))
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"members": out})
}

func (h *TeamHandler) AddMember(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[teamMemberRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.teams.AddMember(c.Ctx(), uid, c.Param("slug"), req.Username)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.Created(c, out)
}

func (h *TeamHandler) RemoveMember(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	if err := h.teams.RemoveMember(c.Ctx(), uid, c.Param("slug"), c.Param("username")); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *TeamHandler) ListRepos(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	out, err := h.teams.ListRepos(c.Ctx(), uid, c.Param("slug"))
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"repos": out})
}

func (h *TeamHandler) SetRepo(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	req, err := httpx.BindJSON[teamRepoRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.teams.SetRepo(c.Ctx(), usecase.SetTeamRepoInput{
		ViewerID: uid,
		TeamSlug: c.Param("slug"),
		RepoName: req.Repo,
		Role:     req.Role,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *TeamHandler) RemoveRepo(c *httpx.Context) error {
	uid, ok := h.viewer(c)
	if !ok {
		return c.Error(http.StatusUnauthorized, "unauthorized")
	}

	if err := h.teams.RemoveRepo(c.Ctx(), uid, c.Param("slug"), c.Param("repo")); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}

func (h *TeamHandler) ListRepoTeams(c *httpx.Context, in usecase.ResolveRepositoryInput) error {
	out, err := h.teams.ListRepoTeams(c.Ctx(), in)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, map[string]any{"teams": out})
}

func (h *TeamHandler) GrantRepoTeam(c *httpx.Context, in usecase.ResolveRepositoryInput) error {
	req, err := httpx.BindJSON[repoTeamRequest](c)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_json")
	}

	out, err := h.teams.GrantRepoTeam(c.Ctx(), usecase.SetRepoTeamInput{
		ResolveRepositoryInput: in,
		TeamSlug:               req.Team,
		Role:                   req.Role,
	})
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, out)
}

func (h *TeamHandler) RevokeRepoTeam(c *httpx.Context, in usecase.ResolveRepositoryInput, teamIDRaw string) error {
	id, err := parseID(teamIDRaw)
	if err != nil {
		return c.Error(http.StatusBadRequest, "invalid_team_id")
	}

	if err := h.teams.RevokeRepoTeam(c.Ctx(), in, id); err != nil {
		return mapError(c, err)
	}

	return c.NoContent()
}
