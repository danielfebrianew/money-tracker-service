package tokens

import (
	"context"
	"time"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/ids"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, userID string) ([]model.APIToken, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Create(ctx context.Context, userID, name string) (*model.APIToken, error) {
	count, err := s.repository.Count(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= 5 {
		return nil, apperror.New(apperror.ErrValidation, "Maksimal 5 API token per akun")
	}
	token := model.APIToken{
		ID:        ids.New("tok"),
		UserID:    userID,
		Token:     ids.Token("ft", 24),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *Service) Delete(ctx context.Context, userID, tokenID string) error {
	return s.repository.Delete(ctx, userID, tokenID)
}
