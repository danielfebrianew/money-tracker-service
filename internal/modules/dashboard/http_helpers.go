package dashboard

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/pkg/apperror"
	"money-management-service/pkg/response"
)

func requireUserID(c echo.Context) (string, error) {
	userID, _ := c.Get("user_id").(string)
	if userID == "" {
		return "", apperror.ErrUnauthorized
	}
	return userID, nil
}

func respondError(c echo.Context, err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		if appErr.Fields != nil {
			return response.ValidationError(c, appErr.Fields)
		}
		message := appErr.Message
		if message == "" {
			message = friendlyMessage(appErr.Err)
		}
		return response.Error(c, statusCode(appErr.Err), message)
	}
	return response.Error(c, statusCode(err), friendlyMessage(err))
}

func statusCode(err error) int {
	switch {
	case errors.Is(err, apperror.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperror.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, apperror.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, apperror.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, apperror.ErrValidation):
		return http.StatusUnprocessableEntity
	case errors.Is(err, apperror.ErrInsufficientFunds):
		return http.StatusPaymentRequired
	case errors.Is(err, apperror.ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, apperror.ErrExternalService):
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func friendlyMessage(err error) string {
	switch {
	case errors.Is(err, apperror.ErrNotFound):
		return "Data tidak ditemukan"
	case errors.Is(err, apperror.ErrUnauthorized):
		return "Tidak terautentikasi"
	case errors.Is(err, apperror.ErrForbidden):
		return "Tidak punya akses"
	case errors.Is(err, apperror.ErrConflict):
		return "Data sudah terdaftar"
	case errors.Is(err, apperror.ErrValidation):
		return "Data tidak valid"
	case errors.Is(err, apperror.ErrInsufficientFunds):
		return "Saldo habis. Silakan top-up."
	case errors.Is(err, apperror.ErrRateLimited):
		return "Terlalu banyak request"
	case errors.Is(err, apperror.ErrExternalService):
		return "Layanan eksternal sedang bermasalah"
	default:
		return "Terjadi kesalahan pada server"
	}
}
