package tokens

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/httphelper"
	"money-management-service/pkg/response"
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
	tokens, err := h.service.List(c.Request().Context(), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, tokens)
}

func (h *Handler) Create(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req CreateRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Name) == "" {
		return response.ValidationError(c, map[string]string{"name": "Nama token wajib diisi"})
	}
	token, err := h.service.Create(c.Request().Context(), userID, req.Name)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, token)
}

func (h *Handler) Delete(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	if err := h.service.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return httphelper.RespondError(c, apperror.New(apperror.ErrNotFound, "Token tidak ditemukan"))
	}
	return response.Message(c, http.StatusOK, "Token berhasil dihapus", nil)
}
