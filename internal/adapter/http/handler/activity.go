package handler

import (
	"strconv"

	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

type ActivityHandler struct {
	activity *usecase.ActivityUseCase
}

func NewActivityHandler(activity *usecase.ActivityUseCase) *ActivityHandler {
	return &ActivityHandler{activity: activity}
}

func (h *ActivityHandler) ListOwner(c *httpx.Context) error {
	var viewerID uint
	if user, ok := middleware.UserFromContext(c.Ctx()); ok {
		viewerID = user.ID
	}

	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.activity.ListOwner(c.Ctx(), c.Param("owner"), viewerID, limit)
	if err != nil {
		return mapError(c, err)
	}

	return httpx.OK(c, activityResponse{Events: items})
}
