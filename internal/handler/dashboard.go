package handler

import (
	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type DashboardHandler struct {
	dashboard *service.DashboardService
}

func NewDashboardHandler(dashboard *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

func (h *DashboardHandler) Summary(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.dashboard.Summary(c.Request().Context(), userID, c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *DashboardHandler) Chart(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.dashboard.Chart(c.Request().Context(), userID, c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *DashboardHandler) Report(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.dashboard.Report(c.Request().Context(), userID, c.QueryParam("period"), c.QueryParam("date"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}
