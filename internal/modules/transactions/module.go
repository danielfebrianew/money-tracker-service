package transactions

import (
	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	"money-management-service/internal/repository"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(store *repository.Store, cache *cache.Cache, parser Parser) *Module {
	repository := NewRepository(store.DB())
	service := NewService(repository, cache, parser)
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
