package groups

import (
	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/httphelper"
	"money-tracker-service/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Create godoc
// @Summary      Buat grup baru
// @Tags         Groups
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body CreateRequest true "Data grup"
// @Success      201 {object} response.Response
// @Router       /groups [post]
func (h *Handler) Create(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req CreateRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	group, members, err := h.service.Create(c.Request().Context(), userID, req.Name)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, map[string]interface{}{"id": group.ID, "name": group.Name, "members": members})
}

// List godoc
// @Summary      Daftar grup user
// @Tags         Groups
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /groups [get]
func (h *Handler) List(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	groups, err := h.service.List(c.Request().Context(), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, groups)
}

// Invite godoc
// @Summary      Undang anggota ke grup
// @Tags         Groups
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string        true "Group ID"
// @Param        body body InviteRequest true "Nomor HP anggota"
// @Success      201 {object} response.Response
// @Failure      404 {object} response.Response
// @Router       /groups/{id}/invite [post]
func (h *Handler) Invite(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req InviteRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	member, err := h.service.Invite(c.Request().Context(), userID, c.Param("id"), req.Phone)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, member)
}

// CreateTransaction godoc
// @Summary      Buat transaksi dalam grup
// @Tags         Groups
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string           true "Group ID"
// @Param        body body TransactionRequest true "Data transaksi"
// @Success      201 {object} response.Response
// @Router       /groups/{id}/transactions [post]
func (h *Handler) CreateTransaction(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req TransactionRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	tx, err := h.service.transactions.CreateGroupTransaction(c.Request().Context(), userID, c.Param("id"), model.CreateTransactionInput{
		Deskripsi: req.Deskripsi,
		Jumlah:    req.Jumlah,
		Kategori:  req.Kategori,
		Tipe:      req.Tipe,
		Source:    "dashboard",
	})
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, tx)
}

// Report godoc
// @Summary      Laporan keuangan grup
// @Tags         Groups
// @Security     BearerAuth
// @Produce      json
// @Param        id    path  string true  "Group ID"
// @Param        month query string false "Bulan (YYYY-MM)"
// @Success      200 {object} response.Response
// @Router       /groups/{id}/report [get]
func (h *Handler) Report(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	report, err := h.service.Report(c.Request().Context(), userID, c.Param("id"), c.QueryParam("month"))
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Success(c, report)
}
