package dashboard

import (
	"github.com/labstack/echo/v4"

	"money-management-service/internal/pkg/httphelper"
	"money-management-service/pkg/response"
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
	data, err := h.service.Summary(c.Request().Context(), userID, c.QueryParam("month"))
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) Chart(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	data, err := h.service.Chart(c.Request().Context(), userID, c.QueryParam("month"))
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) Report(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	data, err := h.service.Report(c.Request().Context(), userID, c.QueryParam("period"), c.QueryParam("date"))
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, data)
}
