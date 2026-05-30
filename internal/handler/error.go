package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"money-tracker-service/pkg/response"
)

func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	message := "Terjadi kesalahan pada server"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if msg, ok := he.Message.(string); ok {
			message = msg
		} else {
			message = http.StatusText(code)
		}
	}

	_ = response.Error(c, code, message)
}
