package groups

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	transactions "money-management-service/internal/modules/transactions"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(db *sqlx.DB, cache *cache.Cache, transactions *transactions.Service) *Module {
	repository := NewRepository(db)
	service := NewService(repository, cache, transactions)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterUserRoutes(api *echo.Group, createMiddlewares []echo.MiddlewareFunc, inviteMiddlewares []echo.MiddlewareFunc) {
	api.POST("/groups", m.Handler.Create, createMiddlewares...)
	api.GET("/groups", m.Handler.List)
	api.POST("/groups/:id/invite", m.Handler.Invite, inviteMiddlewares...)
	api.POST("/groups/:id/transaction", m.Handler.CreateTransaction)
	api.GET("/groups/:id/report", m.Handler.Report)
}
