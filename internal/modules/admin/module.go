package admin

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	"money-management-service/internal/service"
)

type Module struct {
	Handler    *Handler
	Service    *Service
	Repository *Repository
}

func NewModule(auth *service.AuthService, admin *service.AdminService, payments *service.PaymentService, repository *Repository) *Module {
	handler := NewHandler(auth, admin, payments)

	return &Module{
		Handler:    handler,
		Service:    admin,
		Repository: repository,
	}
}

func (m *Module) RegisterRoutes(api *echo.Group, auth *service.AuthService, cache *cache.Cache, adminJWT func(*service.AuthService) echo.MiddlewareFunc, rateLimit func(*cache.Cache, string, int, time.Duration) echo.MiddlewareFunc) {
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
