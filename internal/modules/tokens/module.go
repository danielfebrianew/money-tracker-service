package tokens

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(db *sqlx.DB) *Module {
	repository := NewRepository(db)
	service := NewService(repository)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group, createMiddlewares []echo.MiddlewareFunc, deleteMiddlewares []echo.MiddlewareFunc) {
	api.GET("/tokens", m.Handler.List)
	api.POST("/tokens", m.Handler.Create, createMiddlewares...)
	api.DELETE("/tokens/:id", m.Handler.Delete, deleteMiddlewares...)
}
