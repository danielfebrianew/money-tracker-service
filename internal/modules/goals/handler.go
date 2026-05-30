package goals

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
// @Summary      Daftar target tabungan
// @Tags         Goals
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /goals [get]
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

// Get godoc
// @Summary      Detail target tabungan
// @Tags         Goals
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Goal ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /goals/{id} [get]
func (h *Handler) Get(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	goal, err := h.service.Get(c.Request().Context(), c.Param("id"), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, goal)
}

// Create godoc
// @Summary      Buat target tabungan baru
// @Tags         Goals
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateInput true "Data target"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Router       /goals [post]
func (h *Handler) Create(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req CreateInput
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	goal, err := h.service.Create(c.Request().Context(), userID, req)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusCreated, "Target berhasil dibuat.", goal)
}

// Update godoc
// @Summary      Update target tabungan
// @Tags         Goals
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string      true "Goal ID"
// @Param        body body UpdateInput true "Data yang diupdate"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /goals/{id} [put]
func (h *Handler) Update(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req UpdateInput
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	goal, err := h.service.Update(c.Request().Context(), c.Param("id"), userID, req)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Target berhasil diperbarui.", goal)
}

// Contribute godoc
// @Summary      Tambah atau kurangi kontribusi dana ke target
// @Tags         Goals
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string          true "Goal ID"
// @Param        body body ContributeInput true "Jumlah kontribusi (negatif untuk pengurangan)"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /goals/{id}/contribute [post]
func (h *Handler) Contribute(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req ContributeInput
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	goal, err := h.service.Contribute(c.Request().Context(), c.Param("id"), userID, req)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Kontribusi berhasil ditambahkan.", goal)
}

// Delete godoc
// @Summary      Hapus target tabungan
// @Tags         Goals
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Goal ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /goals/{id} [delete]
func (h *Handler) Delete(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	if err := h.service.Delete(c.Request().Context(), c.Param("id"), userID); err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Target berhasil dihapus.", nil)
}
