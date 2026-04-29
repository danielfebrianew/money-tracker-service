package users

import (
	"github.com/labstack/echo/v4"

	"money-management-service/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Me(c echo.Context) error {
	userID, err := requireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	user, balance, err := h.service.Profile(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, map[string]interface{}{"user": user, "balance": balance})
}

func (h *Handler) UpdateMe(c echo.Context) error {
	userID, err := requireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req UpdateRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := h.service.Update(c.Request().Context(), userID, req.Name, req.Email, req.Timezone)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, user)
}
