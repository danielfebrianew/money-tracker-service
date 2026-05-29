package users

import (
	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/pkg/httphelper"
	"money-tracker-service/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Me(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	user, balance, err := h.service.Profile(c.Request().Context(), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, map[string]interface{}{"user": user, "balance": balance})
}

func (h *Handler) UpdateMe(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req UpdateRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	user, err := h.service.Update(c.Request().Context(), userID, req.Name, req.Email, req.Timezone)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, user)
}
