package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type Handler struct {
	Auth         *service.AuthService
	User         *service.UserService
	Balance      *service.BalanceService
	Tokens       *service.TokenService
	Payments     *service.PaymentService
	Transactions *service.TransactionService
	Dashboard    *service.DashboardService
	Groups       *service.GroupService
	Referral     *service.ReferralService
	Admin        *service.AdminService
	Webhook      *service.WebhookService
}

func RegisterRoutes(e *echo.Echo, h *Handler, auth *service.AuthService, store *repository.Store, cache *cache.Cache) {
	api := e.Group("/api")
	api.GET("/health", h.Health)

	authRate := appmw.RateLimit(cache, "auth", 10, time.Minute)
	api.POST("/auth/register", h.Register, authRate)
	api.POST("/auth/login", h.Login, authRate)
	api.POST("/auth/refresh", h.Refresh, authRate)

	userAPI := api.Group("", appmw.JWT(auth), appmw.RateLimit(cache, "api", 100, time.Minute))
	userAPI.POST("/auth/logout", h.Logout)
	userAPI.GET("/me", h.Me)
	userAPI.PUT("/me", h.UpdateMe)
	userAPI.POST("/me/change-password", h.ChangePassword, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.POST("/transactions", h.CreateTransaction)
	userAPI.GET("/transactions", h.ListTransactions)
	userAPI.GET("/transactions/:id", h.GetTransaction)
	userAPI.DELETE("/transactions/:id", h.DeleteTransaction)
	userAPI.GET("/dashboard/summary", h.DashboardSummary)
	userAPI.GET("/dashboard/chart", h.DashboardChart)
	userAPI.GET("/report", h.Report)
	userAPI.GET("/balance", h.GetBalance)
	userAPI.POST("/payments/topup", h.CreateTopup, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.GET("/payments", h.ListPayments)
	userAPI.GET("/tokens", h.ListTokens)
	userAPI.POST("/tokens", h.CreateToken, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.DELETE("/tokens/:id", h.DeleteToken, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.POST("/groups", h.CreateGroup, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.GET("/groups", h.ListGroups)
	userAPI.POST("/groups/:id/invite", h.InviteGroup, appmw.RateLimit(cache, "auth", 10, time.Minute))
	userAPI.POST("/groups/:id/transaction", h.GroupTransaction)
	userAPI.GET("/groups/:id/report", h.GroupReport)
	userAPI.GET("/referral", h.ReferralSummary)
	userAPI.POST("/referral/generate", h.GenerateReferral, appmw.RateLimit(cache, "auth", 10, time.Minute))

	api.POST("/shortcut", h.Shortcut, appmw.APIToken(store), appmw.RateLimit(cache, "shortcut", 30, time.Minute))
	api.POST("/wa/webhook", h.WAWebhook, appmw.RateLimit(cache, "webhook", 60, time.Minute))

	adminRate := appmw.RateLimit(cache, "admin", 200, time.Minute)
	api.POST("/admin/auth/login", h.AdminLogin, appmw.RateLimit(cache, "admin_auth", 5, time.Minute))
	adminAPI := api.Group("/admin", appmw.AdminJWT(auth), adminRate)
	adminAPI.GET("/dashboard", h.AdminDashboard)
	adminAPI.GET("/users", h.AdminUsers)
	adminAPI.GET("/users/:id", h.AdminUserDetail)
	adminAPI.PUT("/users/:id/status", h.AdminUserStatus)
	adminAPI.PUT("/users/:id/balance", h.AdminUserBalance)
	adminAPI.GET("/payments", h.AdminPayments)
	adminAPI.PUT("/payments/:id/verify", h.AdminVerifyPayment)
	adminAPI.PUT("/payments/:id/reject", h.AdminRejectPayment)
	adminAPI.GET("/revenue", h.AdminRevenue)
	adminAPI.GET("/referrals", h.AdminReferrals)
	adminAPI.POST("/referrals/payout", h.AdminReferralPayout)
	adminAPI.GET("/logs", h.AdminLogs)
}

func (h *Handler) Health(c echo.Context) error {
	return response.Success(c, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (h *Handler) Register(c echo.Context) error {
	var req struct {
		Phone        string  `json:"phone"`
		Name         string  `json:"name"`
		Email        *string `json:"email"`
		Password     string  `json:"password"`
		ReferralCode *string `json:"referral_code"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if errs := validateRegister(req.Phone, req.Name, req.Email, req.Password); len(errs) > 0 {
		return response.ValidationError(c, errs)
	}
	user, balance, pair, err := h.Auth.Register(c.Request().Context(), req.Phone, req.Name, req.Email, req.Password, req.ReferralCode)
	if err != nil {
		return respondError(c, err)
	}
	setAuthCookies(c, pair)
	return response.Created(c, map[string]interface{}{"user": user, "balance": balance, "expires_in": pair.ExpiresIn})
}

func (h *Handler) Login(c echo.Context) error {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	user, balance, pair, err := h.Auth.Login(c.Request().Context(), req.Phone, req.Password)
	if err != nil {
		return respondError(c, err)
	}
	setAuthCookies(c, pair)
	return response.Success(c, map[string]interface{}{"user": user, "balance": balance, "expires_in": pair.ExpiresIn})
}

func (h *Handler) Refresh(c echo.Context) error {
	// Accept refresh token from cookie or request body
	refreshToken := ""
	if cookie, err := c.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}
	if refreshToken == "" {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = c.Bind(&req)
		refreshToken = req.RefreshToken
	}
	if refreshToken == "" {
		return response.Error(c, http.StatusUnauthorized, "Refresh token tidak ditemukan")
	}
	pair, err := h.Auth.Refresh(c.Request().Context(), refreshToken)
	if err != nil {
		return respondError(c, err)
	}
	setAuthCookies(c, pair)
	return response.Success(c, map[string]interface{}{"expires_in": pair.ExpiresIn})
}

func (h *Handler) Logout(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	refreshToken := ""
	if cookie, err := c.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}
	if refreshToken == "" {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = c.Bind(&req)
		refreshToken = req.RefreshToken
	}
	if err := h.Auth.Logout(c.Request().Context(), userID, refreshToken); err != nil {
		return respondError(c, err)
	}
	clearAuthCookies(c)
	return response.Message(c, http.StatusOK, "Berhasil logout", nil)
}

func (h *Handler) Me(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	user, balance, err := h.User.Profile(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, map[string]interface{}{"user": user, "balance": balance})
}

func (h *Handler) UpdateMe(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Timezone *string `json:"timezone"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := h.User.Update(c.Request().Context(), userID, req.Name, req.Email, req.Timezone)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, user)
}

func (h *Handler) ChangePassword(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if req.CurrentPassword == "" || len(req.NewPassword) < 8 || req.CurrentPassword == req.NewPassword {
		return response.ValidationError(c, map[string]string{"new_password": "Minimal 8 karakter dan harus berbeda"})
	}
	if err := h.Auth.ChangePassword(c.Request().Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Password berhasil diubah", nil)
}

func (h *Handler) Shortcut(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Deskripsi string `json:"deskripsi"`
		Jumlah    int    `json:"jumlah"`
		Kategori  string `json:"kategori"`
		Tipe      string `json:"tipe"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if req.Deskripsi == "" || req.Jumlah <= 0 || !validTipe(req.Tipe) {
		return response.ValidationError(c, map[string]string{"request": "deskripsi, jumlah, dan tipe wajib valid"})
	}
	tx, err := h.Transactions.Create(c.Request().Context(), userID, model.CreateTransactionInput{
		Deskripsi: req.Deskripsi,
		Jumlah:    req.Jumlah,
		Kategori:  req.Kategori,
		Tipe:      req.Tipe,
		Source:    "shortcut",
	})
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, tx)
}

func (h *Handler) CreateTransaction(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Deskripsi string `json:"deskripsi"`
		Jumlah    int    `json:"jumlah"`
		Kategori  string `json:"kategori"`
		Tipe      string `json:"tipe"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if req.Deskripsi == "" || req.Jumlah <= 0 || !validTipe(req.Tipe) {
		return response.ValidationError(c, map[string]string{"request": "deskripsi, jumlah, dan tipe wajib valid"})
	}
	tx, err := h.Transactions.Create(c.Request().Context(), userID, model.CreateTransactionInput{
		Deskripsi: req.Deskripsi,
		Jumlah:    req.Jumlah,
		Kategori:  req.Kategori,
		Tipe:      req.Tipe,
		Source:    "dashboard",
	})
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, tx)
}

func (h *Handler) ListTransactions(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	filters, err := transactionFilters(c)
	if err != nil {
		return respondError(c, err)
	}
	items, total, err := h.Transactions.List(c.Request().Context(), userID, filters)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, filters.Page, filters.PerPage)
}

func (h *Handler) GetTransaction(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	tx, err := h.Transactions.Get(c.Request().Context(), userID, c.Param("id"))
	if err != nil {
		return respondError(c, apperror.New(apperror.ErrNotFound, "Transaksi tidak ditemukan"))
	}
	return response.Success(c, tx)
}

func (h *Handler) DeleteTransaction(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	if err := h.Transactions.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Transaksi berhasil dihapus", nil)
}

func (h *Handler) DashboardSummary(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.Dashboard.Summary(c.Request().Context(), userID, c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) DashboardChart(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.Dashboard.Chart(c.Request().Context(), userID, c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) Report(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.Dashboard.Report(c.Request().Context(), userID, c.QueryParam("period"), c.QueryParam("date"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) GetBalance(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	balance, err := h.Balance.GetBalance(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	daysRemaining := 0
	if balance.ExpiresAt != nil {
		daysRemaining = int(time.Until(*balance.ExpiresAt).Hours() / 24)
		if daysRemaining < 0 {
			daysRemaining = 0
		}
	}
	return response.Success(c, map[string]interface{}{
		"balance":         balance.Balance,
		"plan_type":       balance.PlanType,
		"expires_at":      balance.ExpiresAt,
		"days_remaining":  daysRemaining,
		"is_grace_period": false,
	})
}

func (h *Handler) CreateTopup(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	amount, _ := strconv.Atoi(c.FormValue("amount"))
	description := optionalString(c.FormValue("description"))
	proofURL, err := saveProof(c, userID)
	if err != nil {
		return respondError(c, err)
	}
	payment, err := h.Payments.CreateTopup(c.Request().Context(), userID, amount, description, proofURL)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, map[string]interface{}{
		"payment_id": payment.ID,
		"amount":     payment.Amount,
		"status":     payment.Status,
		"message":    "Pembayaran sedang diverifikasi. Estimasi 1x24 jam.",
	})
}

func (h *Handler) ListPayments(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	page, perPage := pagination(c)
	items, total, err := h.Payments.ListUser(c.Request().Context(), userID, c.QueryParam("status"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func (h *Handler) ListTokens(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	tokens, err := h.Tokens.List(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, tokens)
}

func (h *Handler) CreateToken(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Name) == "" {
		return response.ValidationError(c, map[string]string{"name": "Nama token wajib diisi"})
	}
	token, err := h.Tokens.Create(c.Request().Context(), userID, req.Name)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, token)
}

func (h *Handler) DeleteToken(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	if err := h.Tokens.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return respondError(c, apperror.New(apperror.ErrNotFound, "Token tidak ditemukan"))
	}
	return response.Message(c, http.StatusOK, "Token berhasil dihapus", nil)
}

func (h *Handler) WAWebhook(c echo.Context) error {
	var req service.FonnteWebhookPayload
	_ = c.Bind(&req)
	token := c.Request().Header.Get("X-Fonnte-Webhook-Token")
	if token == "" {
		token = c.QueryParam("token")
	}
	go h.Webhook.Handle(context.Background(), req, token)
	return c.NoContent(http.StatusOK)
}

func (h *Handler) CreateGroup(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	group, members, err := h.Groups.Create(c.Request().Context(), userID, req.Name)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, map[string]interface{}{"id": group.ID, "name": group.Name, "members": members})
}

func (h *Handler) ListGroups(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	groups, err := h.Groups.List(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, groups)
}

func (h *Handler) InviteGroup(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Phone string `json:"phone"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	member, err := h.Groups.Invite(c.Request().Context(), userID, c.Param("id"), req.Phone)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, member)
}

func (h *Handler) GroupTransaction(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Deskripsi string `json:"deskripsi"`
		Jumlah    int    `json:"jumlah"`
		Kategori  string `json:"kategori"`
		Tipe      string `json:"tipe"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	tx, err := h.Transactions.CreateGroupTransaction(c.Request().Context(), userID, c.Param("id"), model.CreateTransactionInput{
		Deskripsi: req.Deskripsi,
		Jumlah:    req.Jumlah,
		Kategori:  req.Kategori,
		Tipe:      req.Tipe,
		Source:    "dashboard",
	})
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, tx)
}

func (h *Handler) GroupReport(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	report, err := h.Groups.Report(c.Request().Context(), userID, c.Param("id"), c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, report)
}

func (h *Handler) ReferralSummary(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.Referral.Summary(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) GenerateReferral(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	data, err := h.Referral.Generate(c.Request().Context(), userID)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, data)
}

func (h *Handler) AdminLogin(c echo.Context) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	admin, pair, err := h.Auth.AdminLogin(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, map[string]interface{}{"admin": admin, "access_token": pair.AccessToken, "refresh_token": pair.RefreshToken, "expires_in": pair.ExpiresIn})
}

func (h *Handler) AdminDashboard(c echo.Context) error {
	data, err := h.Admin.Dashboard(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) AdminUsers(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.Admin.ListUsers(c.Request().Context(), c.QueryParam("status"), c.QueryParam("search"), c.QueryParam("sort"), c.QueryParam("order"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func (h *Handler) AdminUserDetail(c echo.Context) error {
	data, err := h.Admin.UserDetail(c.Request().Context(), c.Param("id"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) AdminUserStatus(c echo.Context) error {
	adminID, err := appmw.RequireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		IsActive bool   `json:"is_active"`
		Reason   string `json:"reason"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.Admin.UpdateUserStatus(c.Request().Context(), adminID, c.Param("id"), req.IsActive, req.Reason); err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "User status berhasil diubah", map[string]bool{"is_active": req.IsActive})
}

func (h *Handler) AdminUserBalance(c echo.Context) error {
	adminID, err := appmw.RequireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Amount      int    `json:"amount"`
		Description string `json:"description"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	balance, err := h.Admin.AddUserBalance(c.Request().Context(), adminID, c.Param("id"), req.Amount, req.Description)
	if err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Balance berhasil ditambahkan", map[string]int{"new_balance": balance.Balance})
}

func (h *Handler) AdminPayments(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.Payments.ListAdmin(c.Request().Context(), c.QueryParam("status"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func (h *Handler) AdminVerifyPayment(c echo.Context) error {
	adminID, err := appmw.RequireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	payment, balance, err := h.Payments.Verify(c.Request().Context(), c.Param("id"), adminID)
	if err != nil {
		return respondError(c, err)
	}
	detail := fmt.Sprintf("Verified payment Rp%d", payment.Amount)
	_ = h.Admin.Log(c.Request().Context(), adminID, "verify_payment", strPtr("payment"), &payment.ID, &detail)
	return response.Message(c, http.StatusOK, "Pembayaran berhasil diverifikasi", map[string]interface{}{"payment_id": payment.ID, "new_balance": balance.Balance, "expires_at": balance.ExpiresAt})
}

func (h *Handler) AdminRejectPayment(c echo.Context) error {
	adminID, err := appmw.RequireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)
	if err := h.Payments.Reject(c.Request().Context(), c.Param("id")); err != nil {
		return respondError(c, err)
	}
	paymentID := c.Param("id")
	detail := "Rejected payment. " + req.Reason
	_ = h.Admin.Log(c.Request().Context(), adminID, "reject_payment", strPtr("payment"), &paymentID, &detail)
	return response.Message(c, http.StatusOK, "Pembayaran ditolak", nil)
}

func (h *Handler) AdminRevenue(c echo.Context) error {
	data, err := h.Admin.Revenue(c.Request().Context(), c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) AdminReferrals(c echo.Context) error {
	data, err := h.Admin.ReferralOverview(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) AdminReferralPayout(c echo.Context) error {
	adminID, err := appmw.RequireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		ReferralCode string `json:"referral_code"`
		Period       string `json:"period"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	data, err := h.Admin.CreateReferralPayout(c.Request().Context(), adminID, req.ReferralCode, req.Period)
	if err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Payout berhasil dicatat", data)
}

func (h *Handler) AdminLogs(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.Admin.Logs(c.Request().Context(), c.QueryParam("admin_id"), c.QueryParam("action"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func bind(c echo.Context, dest interface{}) error {
	if err := c.Bind(dest); err != nil {
		return response.Error(c, http.StatusBadRequest, "Request tidak valid")
	}
	return nil
}

func respondError(c echo.Context, err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		if appErr.Fields != nil {
			return response.ValidationError(c, appErr.Fields)
		}
		message := appErr.Message
		if message == "" {
			message = friendlyMessage(appErr.Err)
		}
		return response.Error(c, statusCode(appErr.Err), message)
	}
	return response.Error(c, statusCode(err), friendlyMessage(err))
}

func statusCode(err error) int {
	switch {
	case errors.Is(err, apperror.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperror.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, apperror.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, apperror.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, apperror.ErrValidation):
		return http.StatusUnprocessableEntity
	case errors.Is(err, apperror.ErrInsufficientFunds):
		return http.StatusPaymentRequired
	case errors.Is(err, apperror.ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, apperror.ErrExternalService):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func friendlyMessage(err error) string {
	switch {
	case errors.Is(err, apperror.ErrNotFound):
		return "Data tidak ditemukan"
	case errors.Is(err, apperror.ErrUnauthorized):
		return "Tidak terautentikasi"
	case errors.Is(err, apperror.ErrForbidden):
		return "Tidak punya akses"
	case errors.Is(err, apperror.ErrConflict):
		return "Data sudah terdaftar"
	case errors.Is(err, apperror.ErrValidation):
		return "Data tidak valid"
	case errors.Is(err, apperror.ErrInsufficientFunds):
		return "Saldo habis. Silakan top-up."
	case errors.Is(err, apperror.ErrRateLimited):
		return "Terlalu banyak request"
	case errors.Is(err, apperror.ErrExternalService):
		return "Layanan eksternal sedang bermasalah"
	default:
		return "Terjadi kesalahan pada server"
	}
}

func validateRegister(phone, name string, email *string, password string) map[string]string {
	errs := map[string]string{}
	if !regexp.MustCompile(`^628\d{7,14}$`).MatchString(phone) {
		errs["phone"] = "Format nomor harus 628xxx"
	}
	if len(strings.TrimSpace(name)) < 2 || len(name) > 100 {
		errs["name"] = "Nama minimal 2 karakter dan maksimal 100 karakter"
	}
	if email != nil && *email != "" && !strings.Contains(*email, "@") {
		errs["email"] = "Format email tidak valid"
	}
	if len(password) < 8 {
		errs["password"] = "Minimal 8 karakter"
	}
	return errs
}

func validTipe(tipe string) bool {
	return tipe == "IN" || tipe == "OUT"
}

func pagination(c echo.Context) (int, int) {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func transactionFilters(c echo.Context) (model.TransactionFilters, error) {
	page, perPage := pagination(c)
	filters := model.TransactionFilters{
		Page:     page,
		PerPage:  perPage,
		Tipe:     c.QueryParam("tipe"),
		Kategori: c.QueryParam("kategori"),
		Search:   c.QueryParam("search"),
	}
	if filters.Tipe != "" && !validTipe(filters.Tipe) {
		return filters, apperror.New(apperror.ErrValidation, "Tipe harus IN atau OUT")
	}
	if from := c.QueryParam("from"); from != "" {
		parsed, err := time.Parse("2006-01-02", from)
		if err != nil {
			return filters, apperror.New(apperror.ErrValidation, "Format from harus YYYY-MM-DD")
		}
		filters.From = &parsed
	}
	if to := c.QueryParam("to"); to != "" {
		parsed, err := time.Parse("2006-01-02", to)
		if err != nil {
			return filters, apperror.New(apperror.ErrValidation, "Format to harus YYYY-MM-DD")
		}
		filters.To = &parsed
	}
	return filters, nil
}

func saveProof(c echo.Context, userID string) (*string, error) {
	file, err := c.FormFile("proof")
	if err != nil {
		return nil, nil
	}
	if file.Size > 5*1024*1024 {
		return nil, apperror.New(apperror.ErrValidation, "Ukuran bukti transfer maksimal 5MB")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return nil, apperror.New(apperror.ErrValidation, "Bukti transfer harus jpg atau png")
	}
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	dir := filepath.Join("uploads", "proofs", userID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	name := time.Now().Format("20060102150405") + "_" + ids.RandomHex(4) + ext
	dstPath := filepath.Join(dir, name)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}
	url := "/" + filepath.ToSlash(dstPath)
	return &url, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func strPtr(value string) *string {
	return &value
}

func setAuthCookies(c echo.Context, pair service.TokenPair) {
	secure := isSecure(c)
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    pair.AccessToken,
		Path:     "/",
		MaxAge:   int(pair.ExpiresIn),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	// Refresh token cookie lives 7 days
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    pair.RefreshToken,
		Path:     "/api/auth/refresh",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookies(c echo.Context) {
	secure := isSecure(c)
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func isSecure(c echo.Context) bool {
	return c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https"
}
