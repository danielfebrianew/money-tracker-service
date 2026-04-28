package handler

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/pkg/response"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c echo.Context) error {
	return response.Success(c, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}
