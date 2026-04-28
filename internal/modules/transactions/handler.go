package transactions

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Shortcut(c echo.Context) error {
	return h.create(c, "shortcut")
}

func (h *Handler) Create(c echo.Context) error {
	return h.create(c, "dashboard")
}

func (h *Handler) List(c echo.Context) error {
	userID, err := requireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	filters, err := transactionFilters(c)
	if err != nil {
		return respondError(c, err)
	}
	items, total, err := h.service.List(c.Request().Context(), userID, filters)
	if err != nil {
		return respondError(c, err)
	}
	return response.Paginated(c, items, total, filters.Page, filters.PerPage)
}

func (h *Handler) Get(c echo.Context) error {
	userID, err := requireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	tx, err := h.service.Get(c.Request().Context(), userID, c.Param("id"))
	if err != nil {
		return respondError(c, apperror.New(apperror.ErrNotFound, "Transaksi tidak ditemukan"))
	}
	return response.Success(c, tx)
}

func (h *Handler) Delete(c echo.Context) error {
	userID, err := requireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	if err := h.service.Delete(c.Request().Context(), userID, c.Param("id")); err != nil {
		return respondError(c, err)
	}
	return response.Message(c, http.StatusOK, "Transaksi berhasil dihapus", nil)
}

func (h *Handler) create(c echo.Context, source string) error {
	userID, err := requireUserID(c)
	if err != nil {
		return respondError(c, err)
	}
	var req CreateRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if req.Deskripsi == "" || req.Jumlah <= 0 || !validTipe(req.Tipe) {
		return response.ValidationError(c, map[string]string{"request": "deskripsi, jumlah, dan tipe wajib valid"})
	}
	tx, err := h.service.Create(c.Request().Context(), userID, CreateInput{
		Deskripsi: req.Deskripsi,
		Jumlah:    req.Jumlah,
		Kategori:  req.Kategori,
		Tipe:      req.Tipe,
		Source:    source,
	})
	if err != nil {
		return respondError(c, err)
	}
	return response.Created(c, tx)
}

func validTipe(tipe string) bool {
	return tipe == "IN" || tipe == "OUT"
}

func transactionFilters(c echo.Context) (model.TransactionFilters, error) {
	page, perPage := pagination(c)
	filters := model.TransactionFilters{
		Page:     page,
		PerPage:  perPage,
		Tipe:     c.QueryParam("tipe"),
		Kategori: c.QueryParam("kategori"),
		Search:   c.QueryParam("search"),
	}
	if filters.Tipe != "" && !validTipe(filters.Tipe) {
		return filters, apperror.New(apperror.ErrValidation, "Tipe harus IN atau OUT")
	}
	if from := c.QueryParam("from"); from != "" {
		parsed, err := time.Parse("2006-01-02", from)
		if err != nil {
			return filters, apperror.New(apperror.ErrValidation, "Format from harus YYYY-MM-DD")
		}
		filters.From = &parsed
	}
	if to := c.QueryParam("to"); to != "" {
		parsed, err := time.Parse("2006-01-02", to)
		if err != nil {
			return filters, apperror.New(apperror.ErrValidation, "Format to harus YYYY-MM-DD")
		}
		filters.To = &parsed
	}
	return filters, nil
}
