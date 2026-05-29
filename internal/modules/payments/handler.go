package payments

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/httphelper"
	"money-tracker-service/internal/pkg/ids"
	"money-tracker-service/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateTopup godoc
// @Summary      Ajukan top-up token
// @Tags         Payments
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        amount      formData int    true  "Jumlah top-up"
// @Param        description formData string false "Keterangan"
// @Param        proof       formData file   false "Bukti transfer (maks 5MB)"
// @Success      201 {object} response.Response
// @Failure      400 {object} response.Response
// @Router       /payments/topup [post]
func (h *Handler) CreateTopup(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	amount, _ := strconv.Atoi(c.FormValue("amount"))
	description := optionalString(c.FormValue("description"))
	proofURL, err := saveProof(c, userID)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	payment, err := h.service.CreateTopup(c.Request().Context(), userID, amount, description, proofURL)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	return response.Created(c, TopupResponse{
		PaymentID: payment.ID,
		Amount:    payment.Amount,
		Status:    payment.Status,
		Message:   "Pembayaran sedang diverifikasi. Estimasi 1x24 jam.",
	})
}

// List godoc
// @Summary      Riwayat pembayaran user
// @Tags         Payments
// @Security     BearerAuth
// @Produce      json
// @Param        status   query string false "Filter status: pending / verified / rejected"
// @Param        page     query int    false "Halaman"
// @Param        per_page query int    false "Jumlah per halaman"
// @Success      200 {object} response.Response
// @Router       /payments [get]
func (h *Handler) List(c echo.Context) error {
	userID, err := httphelper.RequireUserID(c)
	if err != nil {
		return httphelper.RespondError(c, err)
	}
	page, perPage := httphelper.Pagination(c)
	items, total, err := h.service.ListUser(c.Request().Context(), userID, c.QueryParam("status"), page, perPage)
	if err != nil {
		return httphelper.RespondError(c, err)
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
