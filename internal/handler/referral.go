package handler

import (
	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type ReferralHandler struct {
	referral *service.ReferralService
}

func NewReferralHandler(referral *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referral: referral}
}

func (h *ReferralHandler) Summary(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.referral.Summary(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *ReferralHandler) Generate(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.referral.Generate(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, data)
}
