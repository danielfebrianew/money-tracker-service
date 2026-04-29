package dashboard

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(cache *cache.Cache, db *sqlx.DB) *Module {
	repository := NewRepository(db)
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
