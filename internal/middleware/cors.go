package middleware

import (
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"money-management-service/internal/config"
)

func CORS(cfg config.Config) echoMiddleware.CORSConfig {
	return echoMiddleware.CORSConfig{
		AllowOrigins:     []string{cfg.AppURL, "http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}
