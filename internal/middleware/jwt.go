package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

func JWT(auth *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenValue := accessToken(c)
			if tokenValue == "" {
				return response.Error(c, http.StatusUnauthorized, "Token tidak ditemukan")
			}
			claims, err := auth.ParseToken(tokenValue, false)
			if err != nil || claims.Type != "access" {
				return response.Error(c, http.StatusUnauthorized, "Token tidak valid atau sudah expired")
			}
			c.Set("user_id", claims.Subject)
			c.Set("role", claims.Role)
			if claims.Role == "admin" || claims.Role == "superadmin" {
				c.Set("admin_id", claims.Subject)
			}
			return next(c)
		}
	}
}

// accessToken reads from HttpOnly cookie first, falls back to Bearer header.
// Cookie is used by Next.js SSR; Bearer is used by Postman and API token clients.
func accessToken(c echo.Context) string {
	if cookie, err := c.Cookie("access_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return bearerToken(c.Request().Header.Get("Authorization"))
}

func AdminJWT(auth *service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return JWT(auth)(func(c echo.Context) error {
			role, _ := c.Get("role").(string)
			if role != "admin" && role != "superadmin" {
				return response.Error(c, http.StatusForbidden, "Akses admin dibutuhkan")
			}
			c.Set("admin_id", c.Get("user_id"))
			return next(c)
		})
	}
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
