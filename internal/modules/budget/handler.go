package budget

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/httphelper"
	"money-tracker-service/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

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

func (h *Handler) History(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	months := 3
	if m := c.QueryParam("months"); m != "" {
		if v, err := strconv.Atoi(m); err == nil {
			months = v
		} else {
			return httphelper.RespondError(c, apperror.New(apperror.ErrValidation, "Parameter months harus berupa angka"))
		}
	}
	items, err := h.service.History(c.Request().Context(), userID, c.QueryParam("kategori"), months)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, items)
}
