package handler

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/service"
)

type WebhookHandler struct {
	webhook *service.WebhookService
}

func NewWebhookHandler(webhook *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{webhook: webhook}
}

func (h *WebhookHandler) WAWebhook(c echo.Context) error {
	var req service.FonnteWebhookPayload
	_ = c.Bind(&req)
	token := c.Request().Header.Get("X-Fonnte-Webhook-Token")
	if token == "" {
		token = c.QueryParam("token")
	}
	go h.webhook.Handle(context.Background(), req, token)
	return c.NoContent(http.StatusOK)
}
