package dashboard

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
// @Summary      Ringkasan keuangan bulanan
// @Tags         Dashboard
// @Security     BearerAuth
// @Produce      json
// @Param        month query string false "Bulan (YYYY-MM), default bulan berjalan"
// @Success      200 {object} response.Response
// @Router       /dashboard/summary [get]
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

// Chart godoc
// @Summary      Data chart bulanan (kategori & tren harian)
// @Tags         Dashboard
// @Security     BearerAuth
// @Produce      json
// @Param        month query string false "Bulan (YYYY-MM)"
// @Success      200 {object} response.Response
// @Router       /dashboard/chart [get]
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

// Report godoc
// @Summary      Laporan keuangan per periode
// @Tags         Dashboard
// @Security     BearerAuth
// @Produce      json
// @Param        period query string false "Periode: weekly / monthly (default monthly)"
// @Param        date   query string false "Tanggal acuan (YYYY-MM-DD)"
// @Success      200 {object} response.Response
// @Router       /dashboard/report [get]
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
