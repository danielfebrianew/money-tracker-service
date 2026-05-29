package accounts

import (
	"context"
	"strings"
	"time"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/ids"
)

var validTypes = map[string]bool{"bank": true, "ewallet": true, "cash": true, "credit_card": true}

var defaultIcon = map[string]string{
	"bank":        "bank",
	"ewallet":     "wallet",
	"cash":        "cash",
	"credit_card": "credit-card",
}

var defaultColor = map[string]string{
	"bank":        "#0066AE",
	"ewallet":     "#00AED6",
	"cash":        "#4CAF50",
	"credit_card": "#9C27B0",
}

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, userID string) ([]model.Account, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, id, userID string) (*model.Account, error) {
	return s.repository.Get(ctx, id, userID)
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (*model.Account, error) {
	name := strings.TrimSpace(input.Name)
	typ := strings.TrimSpace(input.Type)
	if name == "" || len(name) > 100 {
		return nil, apperror.New(apperror.ErrValidation, "Nama akun wajib diisi dan maksimal 100 karakter")
	}
	if !validTypes[typ] {
		return nil, apperror.New(apperror.ErrValidation, "Tipe akun harus bank, ewallet, cash, atau credit_card")
	}
	if input.Balance < 0 {
		return nil, apperror.New(apperror.ErrValidation, "Saldo awal tidak boleh negatif")
	}
	icon := strings.TrimSpace(input.Icon)
	if icon == "" {
		icon = defaultIcon[typ]
	}
	color := strings.TrimSpace(input.Color)
	if color == "" {
		color = defaultColor[typ]
	}
	now := time.Now()
	account := &model.Account{
		ID:        ids.New("acc"),
		UserID:    userID,
		Name:      name,
		Type:      typ,
		Balance:   input.Balance,
		Icon:      icon,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repository.Create(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *Service) Update(ctx context.Context, id, userID string, input UpdateInput) (*model.Account, error) {
	account, err := s.repository.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return nil, apperror.New(apperror.ErrValidation, "Nama akun wajib diisi dan maksimal 100 karakter")
		}
		account.Name = name
	}
	if input.Icon != nil {
		account.Icon = strings.TrimSpace(*input.Icon)
	}
	if input.Color != nil {
		account.Color = strings.TrimSpace(*input.Color)
	}
	account.UpdatedAt = time.Now()
	if err := s.repository.Update(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *Service) Delete(ctx context.Context, id, userID string) error {
	if _, err := s.repository.Get(ctx, id, userID); err != nil {
		return err
	}
	count, err := s.repository.CountTransactions(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperror.New(apperror.ErrConflict, "Akun masih memiliki transaksi, tidak dapat dihapus")
	}
	return s.repository.Delete(ctx, id, userID)
}
