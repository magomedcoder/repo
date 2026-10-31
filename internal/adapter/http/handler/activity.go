package handler

import (
	"strconv"

	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
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
