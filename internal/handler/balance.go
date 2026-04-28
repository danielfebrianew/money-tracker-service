package handler

import (
	"time"

	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type BalanceHandler struct {
	balance *service.BalanceService
}

func NewBalanceHandler(balance *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{balance: balance}
}

func (h *BalanceHandler) Get(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	balance, err := h.balance.GetBalance(c.Request().Context(), userID)
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
	return response.Success(c, map[string]interface{}{
		"balance":         balance.Balance,
		"plan_type":       balance.PlanType,
		"expires_at":      balance.ExpiresAt,
		"days_remaining":  daysRemaining,
		"is_grace_period": false,
	})
}
