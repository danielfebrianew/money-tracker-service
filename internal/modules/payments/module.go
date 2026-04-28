package payments

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

func NewModule(store *repository.Store, cache *cache.Cache) *Module {
	repository := NewRepository(store.DB())
	service := NewService(repository, cache)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group, topupMiddlewares ...echo.MiddlewareFunc) {
	api.POST("/payments/topup", m.Handler.CreateTopup, topupMiddlewares...)
	api.GET("/payments", m.Handler.List)
}
