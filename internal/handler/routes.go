package handler

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	appmw "money-management-service/internal/middleware"
	authmodule "money-management-service/internal/modules/auth"
	"money-management-service/internal/repository"
)

func RegisterRoutes(e *echo.Echo, h *Handler, store *repository.Store, cache *cache.Cache) {
	api := e.Group("/api")
	api.GET("/health", h.Health.Health)

	registerAuthRoutes(api, h.Auth, cache)
	registerExternalRoutes(api, h, store, cache)
	registerUserRoutes(api, h, cache)
	registerAdminRoutes(api, h, cache)
}

func registerAuthRoutes(api *echo.Group, auth *authmodule.Module, cache *cache.Cache) {
	authRate := appmw.RateLimit(cache, "auth", 10, time.Minute)
	auth.RegisterPublicRoutes(api, authRate)
}

func registerUserRoutes(api *echo.Group, h *Handler, cache *cache.Cache) {
	userAPI := api.Group("", appmw.JWT(h.Auth.Service), appmw.RateLimit(cache, "api", 100, time.Minute))
	h.Auth.RegisterUserRoutes(userAPI, appmw.RateLimit(cache, "auth", 10, time.Minute))
	h.User.RegisterUserRoutes(userAPI)

	h.Transactions.RegisterUserRoutes(userAPI)

	h.Dashboard.RegisterUserRoutes(userAPI)
	h.Balance.RegisterUserRoutes(userAPI)

	h.Payments.RegisterUserRoutes(userAPI, appmw.RateLimit(cache, "auth", 10, time.Minute))

	tokenRate := appmw.RateLimit(cache, "auth", 10, time.Minute)
	h.Tokens.RegisterUserRoutes(userAPI, []echo.MiddlewareFunc{tokenRate}, []echo.MiddlewareFunc{tokenRate})

	groupRate := appmw.RateLimit(cache, "auth", 10, time.Minute)
	h.Groups.RegisterUserRoutes(userAPI, []echo.MiddlewareFunc{groupRate}, []echo.MiddlewareFunc{groupRate})

	userAPI.GET("/referral", h.Referral.Summary)
	userAPI.POST("/referral/generate", h.Referral.Generate, appmw.RateLimit(cache, "auth", 10, time.Minute))
}

func registerExternalRoutes(api *echo.Group, h *Handler, store *repository.Store, cache *cache.Cache) {
	h.Transactions.RegisterExternalRoutes(api, appmw.APIToken(store), appmw.RateLimit(cache, "shortcut", 30, time.Minute))
	h.Webhook.RegisterExternalRoutes(api, cache, appmw.RateLimit)
}

func registerAdminRoutes(api *echo.Group, h *Handler, cache *cache.Cache) {
	h.Admin.RegisterRoutes(api, h.Auth.Service, cache, appmw.AdminJWT, appmw.RateLimit)
}
