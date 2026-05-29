package httphelper

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/pkg/response"
)

func Bind(c echo.Context, dest interface{}) error {
	if err := c.Bind(dest); err != nil {
		return response.Error(c, http.StatusBadRequest, "Request tidak valid")
	}
	return nil
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

func Pagination(c echo.Context) (int, int) {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func RespondError(c echo.Context, err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		if appErr.Fields != nil {
			return response.ValidationError(c, appErr.Fields)
		}
		message := appErr.Message
		if message == "" {
			message = FriendlyMessage(appErr.Err)
		}
		return response.Error(c, StatusCode(appErr.Err), message)
	}
	return response.Error(c, StatusCode(err), FriendlyMessage(err))
}

func StatusCode(err error) int {
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

func FriendlyMessage(err error) string {
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
