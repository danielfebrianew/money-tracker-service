package accounts

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
	repo := NewRepository(db)
	svc := NewService(repo)
	handler := NewHandler(svc)
	return &Module{
		Handler:    handler,
		Service:    svc,
		Repository: repo,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group) {
	api.GET("/accounts", m.Handler.List)
	api.POST("/accounts", m.Handler.Create)
	api.GET("/accounts/:id", m.Handler.Get)
	api.PATCH("/accounts/:id", m.Handler.Update)
	api.DELETE("/accounts/:id", m.Handler.Delete)
}
