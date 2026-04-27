package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/repository"
	"money-management-service/pkg/response"
)

func APIToken(store *repository.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tokenValue := bearerToken(c.Request().Header.Get("Authorization"))
			if tokenValue == "" {
				return response.Error(c, http.StatusUnauthorized, "API token tidak ditemukan")
			}
			token, user, err := store.FindAPIToken(c.Request().Context(), tokenValue)
			if err != nil {
				return response.Error(c, http.StatusUnauthorized, "API token tidak valid")
			}
			if !user.IsActive {
				return response.Error(c, http.StatusUnauthorized, "Akun dinonaktifkan")
			}
			store.TouchAPIToken(c.Request().Context(), token.ID)
			c.Set("user_id", user.ID)
			c.Set("user_phone", user.Phone)
			c.Set("role", "user")
			return next(c)
		}
	}
}
