package auth

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/pkg/cookie"
	"money-tracker-service/internal/pkg/httphelper"
	"money-tracker-service/pkg/response"
)

var phonePattern = regexp.MustCompile(`^628\d{7,14}$`)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register godoc
// @Summary      Daftar akun baru
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body RegisterRequest true "Data registrasi"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Failure      409 {object} response.Response
// @Router       /auth/register [post]
func (h *Handler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	if errs := validateRegister(req.Phone, req.Name, req.Email, req.Password); len(errs) > 0 {
		return response.ValidationError(c, errs)
	}
	pair, err := h.service.Register(c.Request().Context(), req.Phone, req.Name, req.Email, req.Password, req.ReferralCode)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	cookie.SetAuthCookies(c, pair, cookie.AudienceUser)
	return response.Created(c, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
	})
}

// Login godoc
// @Summary      Login user
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Kredensial login"
// @Success      200 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /auth/login [post]
func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	_, _, pair, err := h.service.Login(c.Request().Context(), req.Identifier, req.Password)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	cookie.SetAuthCookies(c, pair, cookie.AudienceUser)
	return response.Success(c, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
	})
}

// Refresh godoc
// @Summary      Refresh access token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body RefreshRequest false "Refresh token (opsional jika pakai cookie)"
// @Success      200 {object} response.Response
// @Failure      401 {object} response.Response
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(c echo.Context) error {
	refreshToken := refreshTokenFromRequest(c, cookie.UserRefreshCookie)
	if refreshToken == "" {
		return response.Error(c, http.StatusUnauthorized, "Refresh token tidak ditemukan")
	}
	pair, err := h.service.Refresh(c.Request().Context(), refreshToken)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	cookie.SetAuthCookies(c, pair, cookie.AudienceUser)
	return response.Success(c, map[string]interface{}{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
		"expires_in":    pair.ExpiresIn,
	})
}

// Logout godoc
// @Summary      Logout user
// @Tags         Auth
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} response.Response
// @Router       /auth/logout [post]
func (h *Handler) Logout(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	if err := h.service.Logout(c.Request().Context(), userID, refreshTokenFromRequest(c, cookie.UserRefreshCookie)); err != nil {
		return httphelper.RespondError(c, err)
	}
	cookie.ClearAuthCookies(c, cookie.AudienceUser)
	return response.Message(c, http.StatusOK, "Berhasil logout", nil)
}

// ChangePassword godoc
// @Summary      Ganti password
// @Tags         Auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body ChangePasswordRequest true "Password lama dan baru"
// @Success      200 {object} response.Response
// @Failure      400 {object} response.Response
// @Router       /auth/change-password [put]
func (h *Handler) ChangePassword(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	var req ChangePasswordRequest
	if err := httphelper.Bind(c, &req); err != nil {
		return err
	}
	if req.CurrentPassword == "" || len(req.NewPassword) < 8 || req.CurrentPassword == req.NewPassword {
		return response.ValidationError(c, map[string]string{"new_password": "Minimal 8 karakter dan harus berbeda"})
	}
	if err := h.service.ChangePassword(c.Request().Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Password berhasil diubah", nil)
}

func refreshTokenFromRequest(c echo.Context, cookieName string) string {
	if cookie, err := c.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	var req RefreshRequest
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
