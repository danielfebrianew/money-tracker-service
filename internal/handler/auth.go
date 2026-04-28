package handler

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

var phonePattern = regexp.MustCompile(`^628\d{7,14}$`)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(c echo.Context) error {
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
	user, balance, pair, err := h.auth.Register(c.Request().Context(), req.Phone, req.Name, req.Email, req.Password, req.ReferralCode)
	if err != nil {
		return respondError(c, err)
	}
	setAuthCookies(c, pair)
	return response.Created(c, map[string]interface{}{"user": user, "balance": balance, "expires_in": pair.ExpiresIn})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := bind(c, &req); err != nil {
		return err
	}
	user, balance, pair, err := h.auth.Login(c.Request().Context(), req.Phone, req.Password)
	if err != nil {
		return respondError(c, err)
	}
	setAuthCookies(c, pair)
	return response.Success(c, map[string]interface{}{"user": user, "balance": balance, "expires_in": pair.ExpiresIn})
}

func (h *AuthHandler) Refresh(c echo.Context) error {
	refreshToken := refreshTokenFromRequest(c)
	if refreshToken == "" {
		return response.Error(c, http.StatusUnauthorized, "Refresh token tidak ditemukan")
	}
	pair, err := h.auth.Refresh(c.Request().Context(), refreshToken)
	if err != nil {
		return respondError(c, err)
	}
	setAuthCookies(c, pair)
	return response.Success(c, map[string]interface{}{"expires_in": pair.ExpiresIn})
}

func (h *AuthHandler) Logout(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	if err := h.auth.Logout(c.Request().Context(), userID, refreshTokenFromRequest(c)); err != nil {
		return respondError(c, err)
	}
	clearAuthCookies(c)
	return response.Message(c, http.StatusOK, "Berhasil logout", nil)
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {
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
	if err := h.auth.ChangePassword(c.Request().Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Password berhasil diubah", nil)
}

func refreshTokenFromRequest(c echo.Context) string {
	refreshToken := ""
	if cookie, err := c.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}
	if refreshToken != "" {
		return refreshToken
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.Bind(&req)
	return req.RefreshToken
}

func validateRegister(phone, name string, email *string, password string) map[string]string {
	errs := map[string]string{}
	if !phonePattern.MatchString(phone) {
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
