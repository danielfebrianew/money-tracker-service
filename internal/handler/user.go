package handler

import (
	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) Me(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	user, balance, err := h.users.Profile(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, map[string]interface{}{"user": user, "balance": balance})
}

func (h *UserHandler) UpdateMe(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Timezone *string `json:"timezone"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := h.users.Update(c.Request().Context(), userID, req.Name, req.Email, req.Timezone)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, user)
}
