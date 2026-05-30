package transactions

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"money-tracker-service/internal/cache"
	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/ids"
)

type Parser interface {
	ParseMessage(ctx context.Context, message string) (*model.ParsedTransaction, error)
}

type WalletUpdater interface {
	UpdateBalance(ctx context.Context, tx *sqlx.Tx, walletID string, delta int) error
	Get(ctx context.Context, id, userID string) (*model.Wallet, error)
}

type CategoryValidator interface {
	IsValidForUser(ctx context.Context, userID, name string) (bool, error)
}

type Service struct {
	repository        *Repository
	cache             *cache.Cache
	parser            Parser
	accountUpdater    WalletUpdater
	categoryValidator CategoryValidator
}

func NewService(repository *Repository, cache *cache.Cache, parser Parser, accountUpdater WalletUpdater) *Service {
	return &Service{repository: repository, cache: cache, parser: parser, accountUpdater: accountUpdater}
}

func (s *Service) SetCategoryValidator(v CategoryValidator) {
	s.categoryValidator = v
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (*model.Transaction, error) {
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, apperror.New(apperror.ErrForbidden, "Akun dinonaktifkan. Hubungi admin.")
	}
	tx, err := s.buildTransaction(ctx, user, nil, input, nil)
	if err != nil {
		return nil, err
	}
	if input.WalletID != nil && s.accountUpdater != nil {
		if _, err := s.accountUpdater.Get(ctx, *input.WalletID, userID); err != nil {
			return nil, apperror.New(apperror.ErrNotFound, "Wallet tidak ditemukan")
		}
		dbTx, err := s.repository.BeginTx(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = dbTx.Rollback() }()
		if err := s.repository.CreateTx(ctx, dbTx, tx); err != nil {
			return nil, err
		}
		delta := balanceDelta(tx.Tipe, tx.Jumlah)
		if err := s.accountUpdater.UpdateBalance(ctx, dbTx, *input.WalletID, delta); err != nil {
			return nil, err
		}
		if err := dbTx.Commit(); err != nil {
			return nil, err
		}
	} else {
		if err := s.repository.Create(ctx, tx); err != nil {
			return nil, err
		}
	}
	s.invalidateUserReports(ctx, user.ID)
	return tx, nil
}

func (s *Service) CreateForUser(ctx context.Context, userID, deskripsi string, jumlah int, tipe, source string) (*model.Transaction, error) {
	return s.Create(ctx, userID, CreateInput{
		Jumlah:    jumlah,
		Deskripsi: deskripsi,
		Tipe:      tipe,
		Source:    source,
	})
}

func (s *Service) CreateFromWA(ctx context.Context, user *model.User, parsed *model.ParsedTransaction) (*model.Transaction, error) {
	if err := s.validateUserCanTransact(ctx, user); err != nil {
		return nil, err
	}
	tx, err := s.buildTransaction(ctx, user, nil, CreateInput{
		Jumlah:    parsed.Jumlah,
		Deskripsi: parsed.Deskripsi,
		Kategori:  parsed.Kategori,
		Tipe:      parsed.Tipe,
		Source:    "whatsapp",
	}, &parsed.Confidence)
	if err != nil {
		return nil, err
	}
	if err := s.repository.Create(ctx, tx); err != nil {
		return nil, err
	}
	s.invalidateUserReports(ctx, user.ID)
	return tx, nil
}

func (s *Service) CreateGroupTransaction(ctx context.Context, userID, groupID string, input CreateInput) (*model.Transaction, error) {
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, apperror.New(apperror.ErrForbidden, "Akun dinonaktifkan. Hubungi admin.")
	}
	if _, err := s.repository.GetMembership(ctx, groupID, userID); err != nil {
		return nil, err
	}
	tx, err := s.buildTransaction(ctx, user, &groupID, input, nil)
	if err != nil {
		return nil, err
	}
	if tx.Source == "" {
		tx.Source = "dashboard"
	}
	if err := s.repository.Create(ctx, tx); err != nil {
		return nil, err
	}
	s.cache.DeletePattern(ctx, "group:"+groupID+":report:*")
	return tx, nil
}

func (s *Service) CreateGroupTransactionFromWA(ctx context.Context, user *model.User, groupID string, parsed *model.ParsedTransaction) (*model.Transaction, error) {
	if err := s.validateUserCanTransact(ctx, user); err != nil {
		return nil, err
	}
	tx, err := s.buildTransaction(ctx, user, &groupID, CreateInput{
		Jumlah:    parsed.Jumlah,
		Deskripsi: parsed.Deskripsi,
		Kategori:  parsed.Kategori,
		Tipe:      parsed.Tipe,
		Source:    "whatsapp",
	}, &parsed.Confidence)
	if err != nil {
		return nil, err
	}
	if err := s.repository.Create(ctx, tx); err != nil {
		return nil, err
	}
	s.cache.DeletePattern(ctx, "group:"+groupID+":report:*")
	return tx, nil
}

func (s *Service) List(ctx context.Context, userID string, filters Filters) ([]model.Transaction, int64, error) {
	return s.repository.List(ctx, listParamsFromFilters(userID, nil, filters))
}

func (s *Service) Get(ctx context.Context, userID, txID string) (*model.Transaction, error) {
	return s.repository.Get(ctx, txID, userID)
}

func (s *Service) Delete(ctx context.Context, userID, txID string) error {
	if s.accountUpdater != nil {
		// Fetch first to check if there's an account to reverse balance for
		fetched, err := s.repository.Get(ctx, txID, userID)
		if err != nil {
			return err
		}
		if fetched.WalletID != nil {
			dbTx, err := s.repository.BeginTx(ctx)
			if err != nil {
				return err
			}
			defer func() { _ = dbTx.Rollback() }()
			if err := s.repository.DeleteTx(ctx, dbTx, txID, userID); err != nil {
				return err
			}
			delta := balanceDelta(fetched.Tipe, fetched.Jumlah) * -1
			if err := s.accountUpdater.UpdateBalance(ctx, dbTx, *fetched.WalletID, delta); err != nil {
				return err
			}
			if err := dbTx.Commit(); err != nil {
				return err
			}
			s.invalidateUserReports(ctx, userID)
			return nil
		}
	}
	if _, err := s.repository.Delete(ctx, txID, userID); err != nil {
		return err
	}
	s.invalidateUserReports(ctx, userID)
	return nil
}

func balanceDelta(tipe string, jumlah int) int {
	if tipe == "IN" {
		return jumlah
	}
	return -jumlah
}

func (s *Service) validateUserCanTransact(ctx context.Context, user *model.User) error {
	if !user.IsActive {
		return apperror.New(apperror.ErrForbidden, "Akun dinonaktifkan. Hubungi admin.")
	}
	balance, err := s.repository.GetBalance(ctx, user.ID)
	if err != nil {
		return err
	}
	if balance.Balance <= 0 {
		return apperror.New(apperror.ErrInsufficientFunds, "Saldo habis. Silakan top-up.")
	}
	return nil
}

func (s *Service) buildTransaction(ctx context.Context, user *model.User, groupID *string, input CreateInput, confidence *float64) (*model.Transaction, error) {
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
	if input.Tipe != "IN" && input.Tipe != "OUT" && input.Tipe != "TRANSFER" {
		return nil, apperror.New(apperror.ErrValidation, "Tipe harus IN, OUT, atau TRANSFER")
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
	if s.categoryValidator != nil && kategori != "Lainnya" {
		if valid, _ := s.categoryValidator.IsValidForUser(ctx, user.ID, kategori); !valid {
			kategori = "Lainnya"
		}
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

func (s *Service) invalidateUserReports(ctx context.Context, userID string) {
	s.cache.DeletePattern(ctx, "dash:"+userID+":*")
	s.cache.DeletePattern(ctx, "report:"+userID+":*")
}

func listParamsFromFilters(userID string, groupID *string, filters Filters) ListParams {
	params := ListParams{
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
