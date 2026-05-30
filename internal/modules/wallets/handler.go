package wallets

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
// @Summary      Daftar akun finansial
// @Tags         Accounts
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /accounts [get]
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
// @Summary      Detail akun finansial
// @Tags         Accounts
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /accounts/{id} [get]
func (h *Handler) Get(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	wallet, err := h.service.Get(c.Request().Context(), c.Param("id"), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, wallet)
}

// Create godoc
// @Summary      Buat akun finansial baru
// @Tags         Accounts
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateRequest true "Data akun"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Router       /accounts [post]
func (h *Handler) Create(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req CreateRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	wallet, err := h.service.Create(c.Request().Context(), userID, CreateInput{
		Name:    req.Name,
		Type:    req.Type,
		Balance: req.Balance,
		Icon:    req.Icon,
		Color:   req.Color,
	})
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, wallet)
}

// Update godoc
// @Summary      Update akun finansial
// @Tags         Accounts
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string        true "Account ID"
// @Param        body body UpdateRequest true "Data yang diupdate"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /accounts/{id} [patch]
func (h *Handler) Update(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req UpdateRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	wallet, err := h.service.Update(c.Request().Context(), c.Param("id"), userID, UpdateInput{
		Name:  req.Name,
		Icon:  req.Icon,
		Color: req.Color,
	})
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, wallet)
}

// Delete godoc
// @Summary      Hapus akun finansial
// @Tags         Accounts
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      200 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /accounts/{id} [delete]
func (h *Handler) Delete(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	if err := h.service.Delete(c.Request().Context(), c.Param("id"), userID); err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Wallet berhasil dihapus", nil)
}
