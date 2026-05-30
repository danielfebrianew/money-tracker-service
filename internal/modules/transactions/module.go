package transactions

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/cache"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(db *sqlx.DB, cache *cache.Cache, parser Parser, accountUpdater WalletUpdater) *Module {
	repository := NewRepository(db)
	service := NewService(repository, cache, parser, accountUpdater)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group) {
	api.POST("/transactions", m.Handler.Create)
	api.GET("/transactions", m.Handler.List)
	api.GET("/transactions/:id", m.Handler.Get)
	api.DELETE("/transactions/:id", m.Handler.Delete)
}

func (m *Module) RegisterExternalRoutes(api *echo.Group, middlewares ...echo.MiddlewareFunc) {
	api.POST("/shortcut", m.Handler.Shortcut, middlewares...)
}
