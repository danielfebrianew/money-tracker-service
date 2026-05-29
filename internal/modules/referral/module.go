package referral

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/config"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(cfg config.Config, db *sqlx.DB) *Module {
	repository := NewRepository(db)
	service := NewService(cfg, repository)

	return &Module{
		Handler:    NewHandler(service),
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group, generateMiddlewares ...echo.MiddlewareFunc) {
	api.GET("/referral", m.Handler.Summary)
	api.POST("/referral/generate", m.Handler.Generate, generateMiddlewares...)
}
