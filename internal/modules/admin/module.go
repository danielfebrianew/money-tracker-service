package admin

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
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

	adminRate := rateLimit(cache, "admin", 200, time.Minute)
	adminAPI := api.Group("/admin", adminJWT(auth), adminRate)
	adminAPI.GET("/dashboard", m.Handler.Dashboard)
	adminAPI.GET("/users", m.Handler.Users)
	adminAPI.GET("/users/:id", m.Handler.UserDetail)
	adminAPI.PUT("/users/:id/status", m.Handler.UpdateUserStatus)
	adminAPI.PUT("/users/:id/balance", m.Handler.AddUserBalance)
	adminAPI.GET("/payments", m.Handler.Payments)
	adminAPI.PUT("/payments/:id/verify", m.Handler.VerifyPayment)
	adminAPI.PUT("/payments/:id/reject", m.Handler.RejectPayment)
	adminAPI.GET("/revenue", m.Handler.Revenue)
	adminAPI.GET("/referrals", m.Handler.Referrals)
	adminAPI.POST("/referrals/payout", m.Handler.ReferralPayout)
	adminAPI.GET("/logs", m.Handler.Logs)
}
