package users

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

func (m *Module) RegisterUserRoutes(api *echo.Group) {
	api.GET("/me", m.Handler.Me)
	api.PUT("/me", m.Handler.UpdateMe)
}
