package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/cache"
	"money-management-service/pkg/response"
)

func RateLimit(cache *cache.Cache, group string, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			client := cache.Client()
			if client == nil || limit <= 0 {
				return next(c)
			}
			identifier := c.RealIP()
			if userID, _ := c.Get("user_id").(string); userID != "" {
				identifier = userID
			}
			if adminID, _ := c.Get("admin_id").(string); adminID != "" {
				identifier = adminID
			}
			bucket := time.Now().Unix() / int64(window.Seconds())
			key := "rl:" + identifier + ":" + group + ":" + strconv.FormatInt(bucket, 10)
			count, err := client.Incr(c.Request().Context(), key).Result()
			if err == nil && count == 1 {
				client.Expire(c.Request().Context(), key, window)
			}
			if err == nil && count > int64(limit) {
				c.Response().Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				return response.Error(c, http.StatusTooManyRequests, "Terlalu banyak request. Coba lagi sebentar.")
			}
			return next(c)
		}
	}
}
