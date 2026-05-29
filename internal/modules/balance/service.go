package balance

import (
	"context"
	"time"

	"money-tracker-service/internal/model"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetBalance(ctx context.Context, userID string) (*model.UserBalance, error) {
	return s.repository.Get(ctx, userID)
}

func (s *Service) AddBalance(ctx context.Context, userID string, amount int, expiresAt *time.Time) (*model.UserBalance, error) {
	return s.repository.Add(ctx, userID, amount, expiresAt)
}

func (s *Service) DeductMonthly(ctx context.Context) error {
	return s.repository.DeductMonthly(ctx)
}

func (s *Service) CheckAndSuspend(ctx context.Context) error {
	return s.repository.SuspendExpiredUsers(ctx)
}

func (s *Service) SendExpiryReminders(ctx context.Context) error {
	return nil
}
