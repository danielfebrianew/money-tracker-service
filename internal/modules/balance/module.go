package balance

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

func (m *Module) RegisterUserRoutes(api *echo.Group) {
	api.GET("/balance", m.Handler.Get)
}
