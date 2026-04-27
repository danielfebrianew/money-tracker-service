package service

import (
	"context"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
)

type PaymentService struct {
	store *repository.Store
	cache *cache.Cache
}

func NewPaymentService(store *repository.Store, cache *cache.Cache) *PaymentService {
	return &PaymentService{store: store, cache: cache}
}

func (s *PaymentService) CreateTopup(ctx context.Context, userID string, amount int, description, proofURL *string) (*model.Payment, error) {
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
	if err := s.store.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}
	return &payment, nil
}

func (s *PaymentService) ListUser(ctx context.Context, userID, status string, page, perPage int) ([]model.Payment, int64, error) {
	return s.store.ListUserPayments(ctx, userID, status, page, perPage)
}

func (s *PaymentService) ListAdmin(ctx context.Context, status string, page, perPage int) ([]model.PaymentWithUser, int64, error) {
	return s.store.ListAdminPayments(ctx, status, page, perPage)
}

func (s *PaymentService) Verify(ctx context.Context, paymentID, adminID string) (*model.Payment, *model.UserBalance, error) {
	payment, err := s.store.GetPayment(ctx, paymentID)
	if err != nil {
		return nil, nil, err
	}
	expiresAt := CalculateExpiresAt(payment.Amount)
	verified, balance, err := s.store.VerifyPayment(ctx, paymentID, adminID, expiresAt)
	if err != nil {
		return nil, nil, err
	}
	s.cache.Delete(ctx, "user:"+verified.UserID, "admin:stats")
	return verified, balance, nil
}

func (s *PaymentService) Reject(ctx context.Context, paymentID string) error {
	err := s.store.RejectPayment(ctx, paymentID)
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
