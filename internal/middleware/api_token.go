package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	tokensmodule "money-management-service/internal/modules/tokens"
	"money-management-service/pkg/response"
)

func APIToken(tokens *tokensmodule.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenValue := bearerToken(c.Request().Header.Get("Authorization"))
			if tokenValue == "" {
				return response.Error(c, http.StatusUnauthorized, "API token tidak ditemukan")
			}
			token, user, err := tokens.Find(c.Request().Context(), tokenValue)
			if err != nil {
				return response.Error(c, http.StatusUnauthorized, "API token tidak valid")
			}
			if !user.IsActive {
				return response.Error(c, http.StatusUnauthorized, "Akun dinonaktifkan")
			}
			tokens.Touch(c.Request().Context(), token.ID)
			c.Set("user_id", user.ID)
			c.Set("user_phone", user.Phone)
			c.Set("role", "user")
			return next(c)
		}
	}
}
