package admin

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	appmw "money-management-service/internal/middleware"
	authmodule "money-management-service/internal/modules/auth"
	paymentsmodule "money-management-service/internal/modules/payments"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(auth *authmodule.Service, payments *paymentsmodule.Service, repository *Repository, cache *cache.Cache) *Module {
	service := NewService(repository, cache)
	handler := NewHandler(auth, service, payments)

	return &Module{
		Handler:    handler,
		Service:    service,
		Repository: repository,
	}
}

func (m *Module) RegisterRoutes(api *echo.Group, auth *authmodule.Service, cache *cache.Cache, adminJWT func(*authmodule.Service) echo.MiddlewareFunc, rateLimit func(*cache.Cache, string, int, time.Duration) echo.MiddlewareFunc) {
	api.POST("/admin/auth/login", m.Handler.Login, rateLimit(cache, "admin_auth", 5, time.Minute))
	api.POST("/admin/auth/refresh", m.Handler.Refresh, rateLimit(cache, "admin_auth", 10, time.Minute))

	adminRate := rateLimit(cache, "admin", 200, time.Minute)
	adminAPI := api.Group("/admin", adminJWT(auth), adminRate)
	adminAPI.POST("/auth/logout", m.Handler.Logout)
	adminAPI.GET("/dashboard", m.Handler.Dashboard)
	adminAPI.GET("/users", m.Handler.Users)
	adminAPI.GET("/users/:id", m.Handler.UserDetail)
	adminAPI.PUT("/users/:id/status", m.Handler.UpdateUserStatus, appmw.RequireRole("superadmin"))
	adminAPI.PUT("/users/:id/balance", m.Handler.AddUserBalance, appmw.RequireRole("superadmin"))
	adminAPI.GET("/payments", m.Handler.Payments)
	adminAPI.PUT("/payments/:id/verify", m.Handler.VerifyPayment, appmw.RequireRole("superadmin"))
	adminAPI.PUT("/payments/:id/reject", m.Handler.RejectPayment, appmw.RequireRole("superadmin"))
	adminAPI.GET("/revenue", m.Handler.Revenue)
	adminAPI.GET("/referrals", m.Handler.Referrals)
	adminAPI.POST("/referrals/payout", m.Handler.ReferralPayout, appmw.RequireRole("superadmin"))
	adminAPI.GET("/logs", m.Handler.Logs)
}
