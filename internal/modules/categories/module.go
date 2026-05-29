package categories

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
	api.GET("/categories", m.Handler.List)
	api.POST("/categories", m.Handler.Create)
	api.PUT("/categories/:id", m.Handler.Update)
	api.DELETE("/categories/:id", m.Handler.Delete)
}
