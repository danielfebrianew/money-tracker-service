package webhook

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) WAWebhook(c echo.Context) error {
	var req FonntePayload
	_ = c.Bind(&req)
	token := c.Request().Header.Get("X-Fonnte-Webhook-Token")
	if token == "" {
		token = c.QueryParam("token")
	}
	go h.service.Handle(context.Background(), req, token)
	return c.NoContent(http.StatusOK)
}
