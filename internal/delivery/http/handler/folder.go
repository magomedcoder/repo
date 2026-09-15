package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
)

type FolderHandler struct {
	folders *usecase.FolderUseCase
}

func NewFolderHandler(folders *usecase.FolderUseCase) *FolderHandler {
	return &FolderHandler{folders: folders}
}

type createFolderRequest struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
	Path     string `json:"path"`
}

type renameFolderRequest struct {
	Name string `json:"name"`
}

type moveFolderRequest struct {
	ParentID *uint `json:"parent_id"`
}

func (h *FolderHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.folders.Create(usecase.CreateFolderInput{
		OwnerID:  user.ID,
		Name:     req.Name,
		ParentID: req.ParentID,
		Path:     req.Path,
	})
	if err != nil {
		writeFolderError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, out)
}

func (h *FolderHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tree := r.URL.Query().Get("tree") == "1" || r.URL.Query().Get("tree") == "true"
	var parentID *uint
	if raw := r.URL.Query().Get("parent_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}

		v := uint(id)
		parentID = &v
	}

	out, err := h.folders.List(user.ID, parentID, tree)
	if err != nil {
		writeFolderError(w, err)
		return
	}

	if tree {
		writeJSON(w, http.StatusOK, map[string]any{"tree": out})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"folders": out})
}

func (h *FolderHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	out, err := h.folders.GetContents(user.ID, id)
	if err != nil {
		writeFolderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *FolderHandler) Rename(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	var req renameFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.folders.Rename(usecase.RenameFolderInput{
		OwnerID: user.ID,
		ID:      id,
		Name:    req.Name,
	})
	if err != nil {
		writeFolderError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *FolderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	if err := h.folders.Delete(user.ID, id); err != nil {
		writeFolderError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *FolderHandler) Move(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid folder id")
		return
	}

	var req moveFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	out, err := h.folders.Move(usecase.MoveFolderInput{
		OwnerID:     user.ID,
		ID:          id,
		NewParentID: req.ParentID,
	})
	if err != nil {
		writeFolderError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func parseID(raw string) (uint, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}

	return uint(id), nil
}

func writeFolderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrFolderNameRequired),
		errors.Is(err, usecase.ErrInvalidFolderName),
		errors.Is(err, usecase.ErrInvalidFolderMove):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrFolderExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrFolderNotFound),
		errors.Is(err, usecase.ErrParentFolderNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrFolderNotEmpty),
		errors.Is(err, usecase.ErrFolderDepthExceeded),
		errors.Is(err, usecase.ErrFolderCycle):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
