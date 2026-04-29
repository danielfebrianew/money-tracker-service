package webhook

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	"money-management-service/internal/config"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(cfg config.Config, db *sqlx.DB, cache *cache.Cache, parser Parser, fonnte Sender, transactions TransactionWriter) *Module {
	repository := NewRepository(db)
	service := NewService(cfg, repository, cache, parser, fonnte, transactions)
	return &Module{
		Handler:    NewHandler(service),
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterExternalRoutes(api *echo.Group, cache *cache.Cache, rateLimit func(*cache.Cache, string, int, time.Duration) echo.MiddlewareFunc) {
	api.POST("/wa/webhook", m.Handler.WAWebhook, rateLimit(cache, "webhook", 60, time.Minute))
}
