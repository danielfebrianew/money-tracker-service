package webhook

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(service *Service, repository *Repository) *Module {
	return &Module{
		Handler:    NewHandler(service),
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterExternalRoutes(api *echo.Group, cache *cache.Cache, rateLimit func(*cache.Cache, string, int, time.Duration) echo.MiddlewareFunc) {
	api.POST("/wa/webhook", m.Handler.WAWebhook, rateLimit(cache, "webhook", 60, time.Minute))
}
