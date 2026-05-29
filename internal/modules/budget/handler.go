package budget

import (
	"net/http"

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

// List godoc
// @Summary      Daftar budget per bulan
// @Tags         Budget
// @Security     BearerAuth
// @Produce      json
// @Param        month query string false "Bulan (YYYY-MM), default bulan berjalan"
// @Success      200 {object} response.Response
// @Router       /budgets [get]
func (h *Handler) List(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	items, err := h.service.List(c.Request().Context(), userID, c.QueryParam("month"))
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, items)
}

// Detail godoc
// @Summary      Detail budget beserta rincian transaksi
// @Tags         Budget
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Budget ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /budgets/{id} [get]
func (h *Handler) Detail(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	detail, err := h.service.Detail(c.Request().Context(), userID, c.Param("id"))
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, detail)
}

// Create godoc
// @Summary      Buat budget baru
// @Tags         Budget
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateInput true "Data budget"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      409 {object} response.Response
// @Router       /budgets [post]
func (h *Handler) Create(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req CreateInput
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	budget, err := h.service.Create(c.Request().Context(), userID, req)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusCreated, "Budget berhasil dibuat.", budget)
}

// Update godoc
// @Summary      Update limit budget
// @Tags         Budget
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string      true "Budget ID"
// @Param        body body UpdateInput true "Limit baru"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /budgets/{id} [put]
func (h *Handler) Update(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req UpdateInput
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	budget, err := h.service.Update(c.Request().Context(), userID, c.Param("id"), req.Limit)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Budget berhasil diperbarui.", budget)
}

// Delete godoc
// @Summary      Hapus budget
// @Tags         Budget
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Budget ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /budgets/{id} [delete]
func (h *Handler) Delete(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	if err := h.service.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Budget berhasil dihapus.", nil)
}

