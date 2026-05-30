package goals

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
	api.GET("/goals", m.Handler.List)
	api.POST("/goals", m.Handler.Create)
	api.GET("/goals/:id", m.Handler.Get)
	api.PUT("/goals/:id", m.Handler.Update)
	api.POST("/goals/:id/contribute", m.Handler.Contribute)
	api.DELETE("/goals/:id", m.Handler.Delete)
}
