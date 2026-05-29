package categories

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
// @Summary      Daftar kategori user
// @Tags         Categories
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /categories [get]
func (h *Handler) List(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	items, err := h.service.List(c.Request().Context(), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, items)
}

// Create godoc
// @Summary      Buat kategori baru
// @Tags         Categories
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateInput true "Data kategori"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      409 {object} response.Response
// @Router       /categories [post]
func (h *Handler) Create(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req CreateInput
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	cat, err := h.service.Create(c.Request().Context(), userID, req)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusCreated, "Kategori berhasil dibuat.", cat)
}

// Update godoc
// @Summary      Update kategori
// @Tags         Categories
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string      true "Category ID"
// @Param        body body UpdateInput true "Data yang diupdate"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /categories/{id} [put]
func (h *Handler) Update(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req UpdateInput
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	cat, err := h.service.Update(c.Request().Context(), c.Param("id"), userID, req)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Kategori berhasil diperbarui.", cat)
}

// Delete godoc
// @Summary      Hapus kategori
// @Tags         Categories
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Category ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Failure      409 {object} response.Response
// @Router       /categories/{id} [delete]
func (h *Handler) Delete(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	if err := h.service.Delete(c.Request().Context(), c.Param("id"), userID); err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Kategori berhasil dihapus.", nil)
}
