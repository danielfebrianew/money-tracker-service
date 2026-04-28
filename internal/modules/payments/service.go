package payments

import (
	"context"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
)

type Service struct {
	repository *Repository
	cache      *cache.Cache
}

func NewService(repository *Repository, cache *cache.Cache) *Service {
	return &Service{repository: repository, cache: cache}
}

func (s *Service) CreateTopup(ctx context.Context, userID string, amount int, description, proofURL *string) (*model.Payment, error) {
	if amount < 30000 {
		return nil, apperror.New(apperror.ErrValidation, "Minimal top-up Rp30.000")
	}
	payment := model.Payment{
		ID:          ids.New("pay"),
		UserID:      userID,
		Type:        "topup",
		Amount:      amount,
		Description: description,
		ProofURL:    proofURL,
		Status:      "pending",
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, payment); err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *Service) ListUser(ctx context.Context, userID, status string, page, perPage int) ([]model.Payment, int64, error) {
	return s.repository.ListUser(ctx, userID, status, page, perPage)
}

func (s *Service) ListAdmin(ctx context.Context, status string, page, perPage int) ([]model.PaymentWithUser, int64, error) {
	return s.repository.ListAdmin(ctx, status, page, perPage)
}

func (s *Service) Verify(ctx context.Context, paymentID, adminID string) (*model.Payment, *model.UserBalance, error) {
	payment, err := s.repository.Get(ctx, paymentID)
	if err != nil {
		return nil, nil, err
	}
	expiresAt := CalculateExpiresAt(payment.Amount)
	verified, balance, err := s.repository.Verify(ctx, paymentID, adminID, expiresAt)
	if err != nil {
		return nil, nil, err
	}
	s.cache.Delete(ctx, "user:"+verified.UserID, "admin:stats")
	return verified, balance, nil
}

func (s *Service) Reject(ctx context.Context, paymentID string) error {
	err := s.repository.Reject(ctx, paymentID)
	s.cache.Delete(ctx, "admin:stats")
	return err
}

func CalculateExpiresAt(amount int) *time.Time {
	months := 0
	switch {
	case amount >= 300000:
		months = 12
	case amount >= 160000:
		months = 6
	case amount >= 85000:
		months = 3
	case amount >= 30000:
		months = 1
	}
	if months == 0 {
		return nil
	}
	value := time.Now().UTC().AddDate(0, months, 0)
	return &value
}
