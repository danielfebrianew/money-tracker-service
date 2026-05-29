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

func (h *Handler) Generate(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	data, err := h.service.Generate(c.Request().Context(), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, data)
}
