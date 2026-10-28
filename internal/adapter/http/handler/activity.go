package handler

import (
	"net/http"
	"strconv"

	"repo.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"repo.magomedcoder.ru/repo/internal/usecase"
)

type ActivityHandler struct {
	activity *usecase.ActivityUseCase
}

func NewActivityHandler(activity *usecase.ActivityUseCase) *ActivityHandler {
	return &ActivityHandler{activity: activity}
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
