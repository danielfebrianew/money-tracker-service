package handler

import (
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/repository"
)

func RegisterRoutes(e *echo.Echo, h *Handler, store *repository.Store, cache *cache.Cache) {
	api := e.Group("/api")
	api.GET("/health", h.Health.Health)

	registerAuthRoutes(api, h.Auth, cache)
	registerExternalRoutes(api, h, store, cache)
	registerUserRoutes(api, h, cache)
	registerAdminRoutes(api, h.Admin, cache)
}

func registerAuthRoutes(api *echo.Group, auth *AuthHandler, cache *cache.Cache) {
	authRate := appmw.RateLimit(cache, "auth", 10, time.Minute)
	api.POST("/auth/register", auth.Register, authRate)
	api.POST("/auth/login", auth.Login, authRate)
	api.POST("/auth/refresh", auth.Refresh, authRate)
}

func registerUserRoutes(api *echo.Group, h *Handler, cache *cache.Cache) {
	userAPI := api.Group("", appmw.JWT(h.Auth.auth), appmw.RateLimit(cache, "api", 100, time.Minute))
	userAPI.POST("/auth/logout", h.Auth.Logout)
	userAPI.GET("/me", h.User.Me)
	userAPI.PUT("/me", h.User.UpdateMe)
	userAPI.POST("/me/change-password", h.Auth.ChangePassword, appmw.RateLimit(cache, "auth", 10, time.Minute))

	userAPI.POST("/transactions", h.Transactions.Create)
	userAPI.GET("/transactions", h.Transactions.List)
	userAPI.GET("/transactions/:id", h.Transactions.Get)
	userAPI.DELETE("/transactions/:id", h.Transactions.Delete)

	userAPI.GET("/dashboard/summary", h.Dashboard.Summary)
	userAPI.GET("/dashboard/chart", h.Dashboard.Chart)
	userAPI.GET("/report", h.Dashboard.Report)
	userAPI.GET("/balance", h.Balance.Get)

	userAPI.POST("/payments/topup", h.Payments.CreateTopup, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.GET("/payments", h.Payments.List)

	userAPI.GET("/tokens", h.Tokens.List)
	userAPI.POST("/tokens", h.Tokens.Create, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.DELETE("/tokens/:id", h.Tokens.Delete, appmw.RateLimit(cache, "auth", 10, time.Minute))

	userAPI.POST("/groups", h.Groups.Create, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.GET("/groups", h.Groups.List)
	userAPI.POST("/groups/:id/invite", h.Groups.Invite, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.POST("/groups/:id/transaction", h.Groups.CreateTransaction)
	userAPI.GET("/groups/:id/report", h.Groups.Report)

	userAPI.GET("/referral", h.Referral.Summary)
	userAPI.POST("/referral/generate", h.Referral.Generate, appmw.RateLimit(cache, "auth", 10, time.Minute))
}

func registerExternalRoutes(api *echo.Group, h *Handler, store *repository.Store, cache *cache.Cache) {
	api.POST("/shortcut", h.Transactions.Shortcut, appmw.APIToken(store), appmw.RateLimit(cache, "shortcut", 30, time.Minute))
	api.POST("/wa/webhook", h.Webhook.WAWebhook, appmw.RateLimit(cache, "webhook", 60, time.Minute))
}

func registerAdminRoutes(api *echo.Group, admin *AdminHandler, cache *cache.Cache) {
	api.POST("/admin/auth/login", admin.Login, appmw.RateLimit(cache, "admin_auth", 5, time.Minute))

	adminRate := appmw.RateLimit(cache, "admin", 200, time.Minute)
	adminAPI := api.Group("/admin", appmw.AdminJWT(admin.auth), adminRate)
	adminAPI.GET("/dashboard", admin.Dashboard)
	adminAPI.GET("/users", admin.Users)
	adminAPI.GET("/users/:id", admin.UserDetail)
	adminAPI.PUT("/users/:id/status", admin.UpdateUserStatus)
	adminAPI.PUT("/users/:id/balance", admin.AddUserBalance)
	adminAPI.GET("/payments", admin.Payments)
	adminAPI.PUT("/payments/:id/verify", admin.VerifyPayment)
	adminAPI.PUT("/payments/:id/reject", admin.RejectPayment)
	adminAPI.GET("/revenue", admin.Revenue)
	adminAPI.GET("/referrals", admin.Referrals)
	adminAPI.POST("/referrals/payout", admin.ReferralPayout)
	adminAPI.GET("/logs", admin.Logs)
}
