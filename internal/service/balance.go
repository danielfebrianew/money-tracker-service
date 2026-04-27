package service

import (
	"context"
	"time"

	"money-management-service/internal/model"
	"money-management-service/internal/repository"
)

type BalanceService struct {
	store *repository.Store
}

func NewBalanceService(store *repository.Store) *BalanceService {
	return &BalanceService{store: store}
}

func (s *BalanceService) GetBalance(ctx context.Context, userID string) (*model.UserBalance, error) {
	return s.store.GetBalance(ctx, userID)
}

func (s *BalanceService) AddBalance(ctx context.Context, userID string, amount int, expiresAt *time.Time) (*model.UserBalance, error) {
	return s.store.AddBalance(ctx, userID, amount, expiresAt)
}

func (s *BalanceService) DeductMonthly(ctx context.Context) error {
	_, err := s.store.DB().ExecContext(ctx, `
		UPDATE user_balances
		SET balance = GREATEST(balance - 30000, 0), updated_at = NOW()
		WHERE plan_type = 'monthly' AND balance >= 30000
	`)
	return err
}

func (s *BalanceService) CheckAndSuspend(ctx context.Context) error {
	_, err := s.store.DB().ExecContext(ctx, `
		UPDATE users
		SET is_active = FALSE, updated_at = NOW()
		WHERE id IN (
			SELECT u.id
			FROM users u
			JOIN user_balances b ON b.user_id = u.id
			WHERE u.is_active = TRUE
			  AND b.expires_at IS NOT NULL
			  AND b.expires_at < NOW() - INTERVAL '3 days'
		)
	`)
	return err
}

func (s *BalanceService) SendExpiryReminders(ctx context.Context) error {
	return nil
}
