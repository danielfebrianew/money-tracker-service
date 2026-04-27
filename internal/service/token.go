package service

import (
	"context"
	"time"

	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
)

type TokenService struct {
	store *repository.Store
}

func NewTokenService(store *repository.Store) *TokenService {
	return &TokenService{store: store}
}

func (s *TokenService) List(ctx context.Context, userID string) ([]model.APIToken, error) {
	return s.store.ListAPITokens(ctx, userID)
}

func (s *TokenService) Create(ctx context.Context, userID, name string) (*model.APIToken, error) {
	count, err := s.store.CountAPITokens(ctx, userID)
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
	if err := s.store.CreateAPIToken(ctx, token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *TokenService) Delete(ctx context.Context, userID, tokenID string) error {
	return s.store.DeleteAPIToken(ctx, userID, tokenID)
}
