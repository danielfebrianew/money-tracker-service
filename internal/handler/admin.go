package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type AdminHandler struct {
	auth     *service.AuthService
	admin    *service.AdminService
	payments *service.PaymentService
}

func NewAdminHandler(auth *service.AuthService, admin *service.AdminService, payments *service.PaymentService) *AdminHandler {
	return &AdminHandler{auth: auth, admin: admin, payments: payments}
}

func (h *AdminHandler) Login(c echo.Context) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	admin, pair, err := h.auth.AdminLogin(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, map[string]interface{}{"admin": admin, "access_token": pair.AccessToken, "refresh_token": pair.RefreshToken, "expires_in": pair.ExpiresIn})
}

func (h *AdminHandler) Dashboard(c echo.Context) error {
	data, err := h.admin.Dashboard(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *AdminHandler) Users(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.admin.ListUsers(c.Request().Context(), c.QueryParam("status"), c.QueryParam("search"), c.QueryParam("sort"), c.QueryParam("order"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func (h *AdminHandler) UserDetail(c echo.Context) error {
	data, err := h.admin.UserDetail(c.Request().Context(), c.Param("id"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *AdminHandler) UpdateUserStatus(c echo.Context) error {
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
	if err := h.admin.UpdateUserStatus(c.Request().Context(), adminID, c.Param("id"), req.IsActive, req.Reason); err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "User status berhasil diubah", map[string]bool{"is_active": req.IsActive})
}

func (h *AdminHandler) AddUserBalance(c echo.Context) error {
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
	balance, err := h.admin.AddUserBalance(c.Request().Context(), adminID, c.Param("id"), req.Amount, req.Description)
	if err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Balance berhasil ditambahkan", map[string]int{"new_balance": balance.Balance})
}

func (h *AdminHandler) Payments(c echo.Context) error {
	page, perPage := pagination(c)
	items, total, err := h.payments.ListAdmin(c.Request().Context(), c.QueryParam("status"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func (h *AdminHandler) VerifyPayment(c echo.Context) error {
	adminID, err := appmw.RequireAdminID(c)
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

func (h *AdminHandler) RejectPayment(c echo.Context) error {
	adminID, err := appmw.RequireAdminID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)
	if err := h.payments.Reject(c.Request().Context(), c.Param("id")); err != nil {
		return respondError(c, err)
	}
	paymentID := c.Param("id")
	detail := "Rejected payment. " + req.Reason
	_ = h.admin.Log(c.Request().Context(), adminID, "reject_payment", strPtr("payment"), &paymentID, &detail)
	return response.Message(c, http.StatusOK, "Pembayaran ditolak", nil)
}

func (h *AdminHandler) Revenue(c echo.Context) error {
	data, err := h.admin.Revenue(c.Request().Context(), c.QueryParam("month"))
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *AdminHandler) Referrals(c echo.Context) error {
	data, err := h.admin.ReferralOverview(c.Request().Context())
	if err != nil {
		return respondError(c, err)
	}
	return response.Success(c, data)
}

func (h *AdminHandler) ReferralPayout(c echo.Context) error {
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
	data, err := h.admin.CreateReferralPayout(c.Request().Context(), adminID, req.ReferralCode, req.Period)
	if err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Payout berhasil dicatat", data)
}

func (h *AdminHandler) Logs(c echo.Context) error {
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
