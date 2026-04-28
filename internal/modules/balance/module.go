package balance

import (
	"github.com/labstack/echo/v4"

	"money-management-service/internal/repository"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(store *repository.Store) *Module {
	repository := NewRepository(store.DB())
	service := NewService(repository)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group) {
	api.GET("/balance", m.Handler.Get)
}
