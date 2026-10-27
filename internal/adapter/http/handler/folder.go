package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"git.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"git.magomedcoder.ru/repo/internal/usecase"
)

type FolderHandler struct {
	folders *usecase.FolderUseCase
}

func NewFolderHandler(folders *usecase.FolderUseCase) *FolderHandler {
	return &FolderHandler{folders: folders}
}

type createFolderRequest struct {
	Name string `json:"name"`
}

type renameFolderRequest struct {
	Name string `json:"name"`
}

func (h *FolderHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.folders.Create(usecase.CreateFolderInput{
		ViewerID: user.ID,
		OwnerID:  user.ID,
		Name:     req.Name,
	})
	if err != nil {
		writeFolderError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, out)
}

func (h *FolderHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	out, err := h.folders.List(user.ID, user.ID)
	if err != nil {
		writeFolderError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"folders": out})
}

func (h *FolderHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_folder_id")
		return
	}

	out, err := h.folders.GetContents(user.ID, user.ID, id)
	if err != nil {
		writeFolderError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *FolderHandler) Rename(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_folder_id")
		return
	}

	var req renameFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json")
		return
	}

	out, err := h.folders.Rename(usecase.RenameFolderInput{
		ViewerID: user.ID,
		OwnerID:  user.ID,
		ID:       id,
		Name:     req.Name,
	})

	if err != nil {
		writeFolderError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *FolderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := parseID(r.PathValue("id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_folder_id")
		return
	}

	if err := h.folders.Delete(user.ID, user.ID, id); err != nil {
		writeFolderError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseID(raw string) (uint, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(id), nil
}

func writeFolderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, usecase.ErrFolderNameRequired),
		errors.Is(err, usecase.ErrInvalidFolderName),
		errors.Is(err, usecase.ErrReservedSlug):
		writeError(w, r, http.StatusBadRequest, err.Error())
	case errors.Is(err, usecase.ErrFolderExists):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrFolderNotFound):
		writeError(w, r, http.StatusNotFound, err.Error())
	case errors.Is(err, usecase.ErrFolderNotEmpty):
		writeError(w, r, http.StatusConflict, err.Error())
	case errors.Is(err, usecase.ErrUnauthorized):
		writeError(w, r, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error")
	}
}
