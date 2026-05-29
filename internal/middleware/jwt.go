package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	authmodule "money-tracker-service/internal/modules/auth"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/cookie"
	"money-tracker-service/pkg/response"
)

// UserJWT authenticates a user request. It reads the access token from the
// `user_access_token` cookie, falling back to the Authorization Bearer header.
func UserJWT(auth *authmodule.Service) echo.MiddlewareFunc {
	return jwtMiddleware(auth, cookie.UserAccessCookie, nil)
}

// AdminJWT authenticates an admin request. It reads the access token from the
// `admin_access_token` cookie, falling back to the Authorization Bearer header,
// and rejects any role outside admin/superadmin.
func AdminJWT(auth *authmodule.Service) echo.MiddlewareFunc {
	return jwtMiddleware(auth, cookie.AdminAccessCookie, []string{"admin", "superadmin"})
}

func jwtMiddleware(auth *authmodule.Service, cookieName string, allowedRoles []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenValue := tokenFromRequest(c, cookieName)
			if tokenValue == "" {
				return response.Error(c, http.StatusUnauthorized, "Token tidak ditemukan")
			}
			claims, err := auth.ParseToken(tokenValue, false)
			if err != nil || claims.Type != "access" {
				return response.Error(c, http.StatusUnauthorized, "Token tidak valid atau sudah expired")
			}
			if len(allowedRoles) > 0 && !containsRole(allowedRoles, claims.Role) {
				return response.Error(c, http.StatusForbidden, "Akses admin dibutuhkan")
			}
			c.Set("role", claims.Role)
			if len(allowedRoles) > 0 {
				c.Set("admin_id", claims.Subject)
			} else {
				c.Set("user_id", claims.Subject)
			}
			return next(c)
		}
	}
}

// RequireRole guards an endpoint to a specific subset of roles. It must be
// placed AFTER UserJWT or AdminJWT so the role claim is already in context.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, _ := c.Get("role").(string)
			if !containsRole(roles, role) {
				return response.Error(c, http.StatusForbidden, "Akses tidak diizinkan")
			}
			return next(c)
		}
	}
}

// tokenFromRequest reads the access token from the audience-specific cookie,
// falling back to the Authorization Bearer header. Cookies are scoped per
// audience so user and admin sessions can coexist on the same client.
func tokenFromRequest(c echo.Context, cookieName string) string {
	if cookie, err := c.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return bearerToken(c.Request().Header.Get("Authorization"))
}

func RequireUserID(c echo.Context) (string, error) {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return "", apperror.ErrUnauthorized
	}
	return userID, nil
}

func RequireAdminID(c echo.Context) (string, error) {
	adminID, _ := c.Get("admin_id").(string)
	if adminID == "" {
		return "", apperror.ErrUnauthorized
	}
	return adminID, nil
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func containsRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
