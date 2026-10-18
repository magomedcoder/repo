package handler

import (
	"net/http"
	"strconv"

	"git.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"git.magomedcoder.ru/repo/internal/usecase"
)

type ActivityHandler struct {
	activity *usecase.ActivityUseCase
}

func NewActivityHandler(activity *usecase.ActivityUseCase) *ActivityHandler {
	return &ActivityHandler{activity: activity}
}

func (h *ActivityHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.activity.ListMine(user.ID, limit)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}

func (h *ActivityHandler) ListOwner(w http.ResponseWriter, r *http.Request) {
	var viewerID uint
	if user, ok := middleware.UserFromContext(r.Context()); ok {
		viewerID = user.ID
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.activity.ListOwner(r.PathValue("owner"), viewerID, limit)
	if err != nil {
		if err == usecase.ErrRepoNotFound {
			writeError(w, r, http.StatusNotFound, err.Error())
			return
		}

		writeError(w, r, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}
