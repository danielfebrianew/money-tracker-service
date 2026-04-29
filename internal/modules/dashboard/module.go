package dashboard

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

func NewModule(cache *cache.Cache, store *repository.Store) *Module {
	repository := NewRepository(store.DB())
	service := NewService(cache, repository)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group) {
	api.GET("/dashboard/summary", m.Handler.Summary)
	api.GET("/dashboard/chart", m.Handler.Chart)
	api.GET("/report", m.Handler.Report)
}
