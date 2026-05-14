package admin

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	authmodule "money-management-service/internal/modules/auth"
	paymentsmodule "money-management-service/internal/modules/payments"
	"money-management-service/pkg/response"
)

type Handler struct {
	auth     *authmodule.Service
	admin    *Service
	payments *paymentsmodule.Service
}

func NewHandler(auth *authmodule.Service, admin *Service, payments *paymentsmodule.Service) *Handler {
	return &Handler{auth: auth, admin: admin, payments: payments}
}

func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	admin, pair, err := h.auth.AdminLogin(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return respondError(c, err)
	}
	authmodule.SetAuthCookies(c, pair, authmodule.AudienceAdmin)
	return response.Success(c, map[string]interface{}{"admin": admin, "access_token": pair.AccessToken, "refresh_token": pair.RefreshToken, "expires_in": pair.ExpiresIn})
}

func (h *Handler) Refresh(c echo.Context) error {
	refreshToken := authmodule.RefreshTokenFromRequest(c, authmodule.AdminRefreshCookie)
	if refreshToken == "" {
		return response.Error(c, http.StatusUnauthorized, "Refresh token tidak ditemukan")
	}
	pair, err := h.auth.AdminRefresh(c.Request().Context(), refreshToken)
	if err != nil {
		return respondError(c, err)
	}
	authmodule.SetAuthCookies(c, pair, authmodule.AudienceAdmin)
	return response.Success(c, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
	})
}

func (h *Handler) Logout(c echo.Context) error {
	refreshToken := authmodule.RefreshTokenFromRequest(c, authmodule.AdminRefreshCookie)
	_ = h.auth.AdminLogout(c.Request().Context(), refreshToken)
	authmodule.ClearAuthCookies(c, authmodule.AudienceAdmin)
	return response.Message(c, http.StatusOK, "Berhasil logout", nil)
}

func (h *Handler) Dashboard(c echo.Context) error {
	data, err := h.admin.Dashboard(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) Users(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.admin.ListUsers(c.Request().Context(), c.QueryParam("status"), c.QueryParam("search"), c.QueryParam("sort"), c.QueryParam("order"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func (h *Handler) UserDetail(c echo.Context) error {
	data, err := h.admin.UserDetail(c.Request().Context(), c.Param("id"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) UpdateUserStatus(c echo.Context) error {
	adminID, err := requireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req UpdateUserStatusRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := h.admin.UpdateUserStatus(c.Request().Context(), adminID, c.Param("id"), req.IsActive, req.Reason); err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "User status berhasil diubah", map[string]bool{"is_active": req.IsActive})
}

func (h *Handler) AddUserBalance(c echo.Context) error {
	adminID, err := requireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req AddUserBalanceRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	balance, err := h.admin.AddUserBalance(c.Request().Context(), adminID, c.Param("id"), req.Amount, req.Description)
	if err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Balance berhasil ditambahkan", map[string]int{"new_balance": balance.Balance})
}

func (h *Handler) Payments(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.payments.ListAdmin(c.Request().Context(), c.QueryParam("status"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func (h *Handler) VerifyPayment(c echo.Context) error {
	adminID, err := requireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	payment, balance, err := h.payments.Verify(c.Request().Context(), c.Param("id"), adminID)
	if err != nil {
		return respondError(c, err)
	}
	detail := fmt.Sprintf("Verified payment Rp%d", payment.Amount)
	_ = h.admin.Log(c.Request().Context(), adminID, "verify_payment", strPtr("payment"), &payment.ID, &detail)
	return response.Message(c, http.StatusOK, "Pembayaran berhasil diverifikasi", map[string]interface{}{"payment_id": payment.ID, "new_balance": balance.Balance, "expires_at": balance.ExpiresAt})
}

func (h *Handler) RejectPayment(c echo.Context) error {
	adminID, err := requireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req RejectPaymentRequest
	_ = c.Bind(&req)
	if err := h.payments.Reject(c.Request().Context(), c.Param("id")); err != nil {
		return respondError(c, err)
	}
	paymentID := c.Param("id")
	detail := "Rejected payment. " + req.Reason
	_ = h.admin.Log(c.Request().Context(), adminID, "reject_payment", strPtr("payment"), &paymentID, &detail)
	return response.Message(c, http.StatusOK, "Pembayaran ditolak", nil)
}

func (h *Handler) Revenue(c echo.Context) error {
	data, err := h.admin.Revenue(c.Request().Context(), c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) Referrals(c echo.Context) error {
	data, err := h.admin.ReferralOverview(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *Handler) ReferralPayout(c echo.Context) error {
	adminID, err := requireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req ReferralPayoutRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	data, err := h.admin.CreateReferralPayout(c.Request().Context(), adminID, req.ReferralCode, req.Period)
	if err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Payout berhasil dicatat", data)
}

func (h *Handler) Logs(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.admin.Logs(c.Request().Context(), c.QueryParam("admin_id"), c.QueryParam("action"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func strPtr(value string) *string {
	return &value
}
