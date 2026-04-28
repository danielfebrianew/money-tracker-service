package handler

import (
	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/model"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type GroupHandler struct {
	groups       *service.GroupService
	transactions *service.TransactionService
}

func NewGroupHandler(groups *service.GroupService, transactions *service.TransactionService) *GroupHandler {
	return &GroupHandler{groups: groups, transactions: transactions}
}

func (h *GroupHandler) Create(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	group, members, err := h.groups.Create(c.Request().Context(), userID, req.Name)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, map[string]interface{}{"id": group.ID, "name": group.Name, "members": members})
}

func (h *GroupHandler) List(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	groups, err := h.groups.List(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, groups)
}

func (h *GroupHandler) Invite(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Phone string `json:"phone"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	member, err := h.groups.Invite(c.Request().Context(), userID, c.Param("id"), req.Phone)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, member)
}

func (h *GroupHandler) CreateTransaction(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Deskripsi string `json:"deskripsi"`
		Jumlah    int    `json:"jumlah"`
		Kategori  string `json:"kategori"`
		Tipe      string `json:"tipe"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	tx, err := h.transactions.CreateGroupTransaction(c.Request().Context(), userID, c.Param("id"), model.CreateTransactionInput{
		Deskripsi: req.Deskripsi,
		Jumlah:    req.Jumlah,
		Kategori:  req.Kategori,
		Tipe:      req.Tipe,
		Source:    "dashboard",
	})
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, tx)
}

func (h *GroupHandler) Report(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	report, err := h.groups.Report(c.Request().Context(), userID, c.Param("id"), c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, report)
}
