package auth

import (
	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	"money-management-service/internal/config"
	"money-management-service/internal/repository"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(cfg config.Config, store *repository.Store, cache *cache.Cache) *Module {
	repository := NewRepository(store.DB())
	service := NewService(cfg, repository, cache)
	handler := NewHandler(service)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterPublicRoutes(api *echo.Group, middlewares ...echo.MiddlewareFunc) {
	api.POST("/auth/register", m.Handler.Register, middlewares...)
	api.POST("/auth/login", m.Handler.Login, middlewares...)
	api.POST("/auth/refresh", m.Handler.Refresh, middlewares...)
}

func (m *Module) RegisterUserRoutes(api *echo.Group, changePasswordMiddlewares ...echo.MiddlewareFunc) {
	api.POST("/auth/logout", m.Handler.Logout)
	api.POST("/me/change-password", m.Handler.ChangePassword, changePasswordMiddlewares...)
}
