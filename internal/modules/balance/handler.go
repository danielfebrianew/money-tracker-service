package balance

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Get(c echo.Context) error {
	userID, err := requireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	balance, err := h.service.GetBalance(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	daysRemaining := 0
	if balance.ExpiresAt != nil {
		daysRemaining = int(time.Until(*balance.ExpiresAt).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}
	}
	return response.Success(c, Response{
		Balance:       balance.Balance,
		PlanType:      balance.PlanType,
		ExpiresAt:     balance.ExpiresAt,
		DaysRemaining: daysRemaining,
		IsGracePeriod: false,
	})
}
