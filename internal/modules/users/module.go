package users

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

func NewModule(db *sqlx.DB, cache *cache.Cache) *Module {
	repository := NewRepository(db)
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
