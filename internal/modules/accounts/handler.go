package accounts

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

func (h *Handler) Get(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	account, err := h.service.Get(c.Request().Context(), c.Param("id"), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, account)
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
	account, err := h.service.Create(c.Request().Context(), userID, CreateInput{
		Name: req.Name,
		Type: req.Type,
	})
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, account)
}

func (h *Handler) Update(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req UpdateRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	account, err := h.service.Update(c.Request().Context(), c.Param("id"), userID, UpdateInput{
		Name: req.Name,
		Type: req.Type,
	})
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, account)
}

func (h *Handler) Delete(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	if err := h.service.Delete(c.Request().Context(), c.Param("id"), userID); err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Akun berhasil dihapus", nil)
}
