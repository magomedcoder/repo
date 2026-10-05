package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

type OrganizationHandler struct {
	orgs *usecase.OrganizationUseCase
}

func NewOrganizationHandler(orgs *usecase.OrganizationUseCase) *OrganizationHandler {
	return &OrganizationHandler{orgs: orgs}
}

type createOrgRequest struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateOrgRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type orgMemberRequest struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type updateMemberRequest struct {
	Role string `json:"role"`
}

func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.orgs.Create(usecase.CreateOrganizationInput{
		CreatorID:   user.ID,
		Slug:        req.Slug,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, out)
}

func (h *OrganizationHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.orgs.ListMine(user.ID)
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"organizations": items})
}

func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	var viewerID uint
	if user, ok := middleware.UserFromContext(r.Context()); ok {
		viewerID = user.ID
	}

	out, err := h.orgs.Get(r.PathValue("slug"), viewerID)
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *OrganizationHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.orgs.Update(usecase.UpdateOrganizationInput{
		ViewerID:    user.ID,
		Slug:        r.PathValue("slug"),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *OrganizationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.orgs.Delete(r.PathValue("slug"), user.ID); err != nil {
		writeOrgError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrganizationHandler) ListRepos(w http.ResponseWriter, r *http.Request) {
	var viewerID uint
	if user, ok := middleware.UserFromContext(r.Context()); ok {
		viewerID = user.ID
	}

	items, err := h.orgs.ListRepos(r.PathValue("slug"), viewerID)
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"repos": items})
}

func (h *OrganizationHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.orgs.ListMembers(r.PathValue("slug"), user.ID)
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"members": items})
}

func (h *OrganizationHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req orgMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.orgs.AddMember(r.PathValue("slug"), user.ID, req.Username, req.Role)
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, out)
}

func (h *OrganizationHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	uid, err := strconv.ParseUint(r.PathValue("userID"), 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_user_id")
		return
	}

	var req updateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.orgs.UpdateMemberRole(r.PathValue("slug"), user.ID, uint(uid), req.Role)
	if err != nil {
		writeOrgError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *OrganizationHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	
	uid, err := strconv.ParseUint(r.PathValue("userID"), 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_user_id")
		return
	}

	if err := h.orgs.RemoveMember(r.PathValue("slug"), user.ID, uint(uid)); err != nil {
		writeOrgError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrganizationHandler) Leave(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	
	if err := h.orgs.Leave(r.PathValue("slug"), user.ID); err != nil {
		writeOrgError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeOrgError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrOrgSlugRequired),
		errors.Is(err, usecase.ErrOrgNameRequired),
		errors.Is(err, usecase.ErrInvalidUsername),
		errors.Is(err, usecase.ErrInvalidOrgRole),
		errors.Is(err, usecase.ErrReservedSlug):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrOrgSlugTaken),
		errors.Is(err, usecase.ErrOrgMemberExists),
		errors.Is(err, usecase.ErrCannotRemoveOwner),
		errors.Is(err, usecase.ErrOrgNotEmpty):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrOrgNotFound),
		errors.Is(err, usecase.ErrOrgMemberNotFound),
		errors.Is(err, usecase.ErrUserNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrOrgForbidden):
		writeError(w, r, http.StatusForbidden, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
