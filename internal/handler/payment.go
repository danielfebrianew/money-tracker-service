package handler

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	appmw "money-management-service/internal/middleware"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/service"
	"money-management-service/pkg/response"
)

type PaymentHandler struct {
	payments *service.PaymentService
}

func NewPaymentHandler(payments *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

func (h *PaymentHandler) CreateTopup(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	amount, _ := strconv.Atoi(c.FormValue("amount"))
	description := optionalString(c.FormValue("description"))
	proofURL, err := saveProof(c, userID)
	if err != nil {
		return respondError(c, err)
	}
	payment, err := h.payments.CreateTopup(c.Request().Context(), userID, amount, description, proofURL)
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, map[string]interface{}{
		"payment_id": payment.ID,
		"amount":     payment.Amount,
		"status":     payment.Status,
		"message":    "Pembayaran sedang diverifikasi. Estimasi 1x24 jam.",
	})
}

func (h *PaymentHandler) List(c echo.Context) error {
	userID, err := appmw.RequireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	page, perPage := pagination(c)
	items, total, err := h.payments.ListUser(c.Request().Context(), userID, c.QueryParam("status"), page, perPage)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, page, perPage)
}

func saveProof(c echo.Context, userID string) (*string, error) {
	file, err := c.FormFile("proof")
	if err != nil {
		return nil, nil
	}
	if file.Size > 5*1024*1024 {
		return nil, apperror.New(apperror.ErrValidation, "Ukuran bukti transfer maksimal 5MB")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return nil, apperror.New(apperror.ErrValidation, "Bukti transfer harus jpg atau png")
	}
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	dir := filepath.Join("uploads", "proofs", userID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	name := time.Now().Format("20060102150405") + "_" + ids.RandomHex(4) + ext
	dstPath := filepath.Join(dir, name)
	dst, err := os.Create(dstPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}
	url := "/" + filepath.ToSlash(dstPath)
	return &url, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
