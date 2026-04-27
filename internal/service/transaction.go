package service

import (
	"context"
	"strings"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
)

type TransactionService struct {
	store  *repository.Store
	cache  *cache.Cache
	parser OpenAIService
}

func NewTransactionService(store *repository.Store, cache *cache.Cache, parser OpenAIService) *TransactionService {
	return &TransactionService{store: store, cache: cache, parser: parser}
}

func (s *TransactionService) Create(ctx context.Context, userID string, input model.CreateTransactionInput) (*model.Transaction, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.validateUserCanTransact(ctx, user); err != nil {
		return nil, err
	}
	tx, err := s.buildTransaction(ctx, user, nil, input, nil)
	if err != nil {
		return nil, err
	}
	if err := s.store.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}
	s.invalidateUserReports(ctx, user.ID)
	return tx, nil
}

func (s *TransactionService) CreateForUser(ctx context.Context, userID, deskripsi string, jumlah int, tipe, source string) (*model.Transaction, error) {
	return s.Create(ctx, userID, model.CreateTransactionInput{
		Jumlah:    jumlah,
		Deskripsi: deskripsi,
		Tipe:      tipe,
		Source:    source,
	})
}

func (s *TransactionService) CreateFromWA(ctx context.Context, user *model.User, parsed *model.ParsedTransaction) (*model.Transaction, error) {
	if err := s.validateUserCanTransact(ctx, user); err != nil {
		return nil, err
	}
	tx, err := s.buildTransaction(ctx, user, nil, model.CreateTransactionInput{
		Jumlah:    parsed.Jumlah,
		Deskripsi: parsed.Deskripsi,
		Kategori:  parsed.Kategori,
		Tipe:      parsed.Tipe,
		Source:    "whatsapp",
	}, &parsed.Confidence)
	if err != nil {
		return nil, err
	}
	if err := s.store.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}
	s.invalidateUserReports(ctx, user.ID)
	return tx, nil
}

func (s *TransactionService) CreateGroupTransaction(ctx context.Context, userID, groupID string, input model.CreateTransactionInput) (*model.Transaction, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.validateUserCanTransact(ctx, user); err != nil {
		return nil, err
	}
	if _, err := s.store.GetMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}
	tx, err := s.buildTransaction(ctx, user, &groupID, input, nil)
	if err != nil {
		return nil, err
	}
	if tx.Source == "" {
		tx.Source = "dashboard"
	}
	if err := s.store.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}
	s.cache.DeletePattern(ctx, "group:"+groupID+":report:*")
	return tx, nil
}

func (s *TransactionService) CreateGroupTransactionFromWA(ctx context.Context, user *model.User, groupID string, parsed *model.ParsedTransaction) (*model.Transaction, error) {
	if err := s.validateUserCanTransact(ctx, user); err != nil {
		return nil, err
	}
	tx, err := s.buildTransaction(ctx, user, &groupID, model.CreateTransactionInput{
		Jumlah:    parsed.Jumlah,
		Deskripsi: parsed.Deskripsi,
		Kategori:  parsed.Kategori,
		Tipe:      parsed.Tipe,
		Source:    "whatsapp",
	}, &parsed.Confidence)
	if err != nil {
		return nil, err
	}
	if err := s.store.CreateTransaction(ctx, tx); err != nil {
		return nil, err
	}
	s.cache.DeletePattern(ctx, "group:"+groupID+":report:*")
	return tx, nil
}

func (s *TransactionService) List(ctx context.Context, userID string, filters model.TransactionFilters) ([]model.Transaction, int64, error) {
	return s.store.ListTransactions(ctx, listParamsFromFilters(userID, nil, filters))
}

func (s *TransactionService) Get(ctx context.Context, userID, txID string) (*model.Transaction, error) {
	return s.store.GetTransaction(ctx, txID, userID)
}

func (s *TransactionService) Delete(ctx context.Context, userID, txID string) error {
	err := s.store.DeleteTransaction(ctx, txID, userID)
	if err == nil {
		s.invalidateUserReports(ctx, userID)
	}
	return err
}

func (s *TransactionService) Summary(ctx context.Context, userID, month string) (*model.DashboardSummary, error) {
	return s.store.GetTransactionSummary(ctx, userID, month)
}

func (s *TransactionService) ChartData(ctx context.Context, userID, month, timezone string) (*model.ChartData, error) {
	return s.store.GetTransactionChartData(ctx, userID, month, timezone)
}

func (s *TransactionService) validateUserCanTransact(ctx context.Context, user *model.User) error {
	if !user.IsActive {
		return apperror.New(apperror.ErrForbidden, "Akun dinonaktifkan. Hubungi admin.")
	}
	balance, err := s.store.GetBalance(ctx, user.ID)
	if err != nil {
		return err
	}
	if balance.Balance <= 0 {
		return apperror.New(apperror.ErrInsufficientFunds, "Saldo habis. Silakan top-up.")
	}
	return nil
}

func (s *TransactionService) buildTransaction(ctx context.Context, user *model.User, groupID *string, input model.CreateTransactionInput, confidence *float64) (*model.Transaction, error) {
	input.Deskripsi = strings.TrimSpace(input.Deskripsi)
	input.Tipe = strings.ToUpper(strings.TrimSpace(input.Tipe))
	if input.Source == "" {
		input.Source = "dashboard"
	}
	if input.Jumlah <= 0 {
		return nil, apperror.New(apperror.ErrValidation, "Jumlah transaksi wajib lebih dari 0")
	}
	if input.Deskripsi == "" || len(input.Deskripsi) > 255 {
		return nil, apperror.New(apperror.ErrValidation, "Deskripsi wajib diisi dan maksimal 255 karakter")
	}
	if input.Tipe != "IN" && input.Tipe != "OUT" {
		return nil, apperror.New(apperror.ErrValidation, "Tipe harus IN atau OUT")
	}
	kategori := strings.TrimSpace(input.Kategori)
	if kategori == "" {
		parsed, _ := s.parser.ParseMessage(ctx, input.Deskripsi+" 1000")
		if parsed != nil && parsed.Kategori != "" {
			kategori = parsed.Kategori
		}
	}
	if kategori == "" {
		kategori = "Lainnya"
	}
	if len(kategori) > 50 {
		kategori = kategori[:50]
	}
	recordedBy := user.Phone
	return &model.Transaction{
		ID:         ids.New("txn"),
		UserID:     user.ID,
		GroupID:    groupID,
		Jumlah:     input.Jumlah,
		Deskripsi:  input.Deskripsi,
		Kategori:   kategori,
		Tipe:       input.Tipe,
		Source:     input.Source,
		RecordedBy: &recordedBy,
		Confidence: confidence,
		CreatedAt:  time.Now().In(userLocation(user.Timezone)),
	}, nil
}

func (s *TransactionService) invalidateUserReports(ctx context.Context, userID string) {
	s.cache.DeletePattern(ctx, "dash:"+userID+":*")
	s.cache.DeletePattern(ctx, "report:"+userID+":*")
}

func listParamsFromFilters(userID string, groupID *string, filters model.TransactionFilters) model.TransactionListParams {
	params := model.TransactionListParams{
		UserID:  userID,
		GroupID: groupID,
		From:    filters.From,
		To:      filters.To,
		Page:    filters.Page,
		PerPage: filters.PerPage,
	}
	if filters.To != nil {
		to := filters.To.Add(24 * time.Hour)
		params.To = &to
	}
	if filters.Tipe != "" {
		params.Tipe = &filters.Tipe
	}
	if filters.Kategori != "" {
		params.Kategori = &filters.Kategori
	}
	if filters.Search != "" {
		params.Search = &filters.Search
	}
	return params
}

func userLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}
