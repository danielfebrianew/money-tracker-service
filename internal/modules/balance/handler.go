package balance

import (
	"time"

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

func (h *Handler) Get(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	balance, err := h.service.GetBalance(c.Request().Context(), userID)
	if err != nil {
		return httphelper.RespondError(c, err)
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
