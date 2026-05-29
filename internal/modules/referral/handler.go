package referral

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

// Summary godoc
// @Summary      Ringkasan referral user
// @Tags         Referral
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /referral/summary [get]
func (h *Handler) Summary(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	data, err := h.service.Summary(c.Request().Context(), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, data)
}

