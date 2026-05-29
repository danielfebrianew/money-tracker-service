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
