package users

import (
	"context"
	"time"

	"money-tracker-service/internal/cache"
	"money-tracker-service/internal/model"
)

type Service struct {
	repository *Repository
	cache      *cache.Cache
}

func NewService(repository *Repository, cache *cache.Cache) *Service {
	return &Service{repository: repository, cache: cache}
}

func (s *Service) Profile(ctx context.Context, userID string) (*model.User, *model.UserBalance, error) {
	var cached struct {
		User    *model.User        `json:"user"`
		Balance *model.UserBalance `json:"balance"`
	}
	if s.cache.GetJSON(ctx, "user:"+userID, &cached) && cached.User != nil && cached.Balance != nil {
		return cached.User, cached.Balance, nil
	}
	user, err := s.repository.GetByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	balance, err := s.repository.GetBalance(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	s.cache.SetJSON(ctx, "user:"+userID, map[string]interface{}{"user": user, "balance": balance}, 15*time.Minute)
	return user, balance, nil
}

func (s *Service) Update(ctx context.Context, userID string, name, email, timezone *string) (*model.User, error) {
	user, err := s.repository.Update(ctx, userID, name, email, timezone)
	if err != nil {
		return nil, err
	}
	s.cache.Delete(ctx, "user:"+userID)
	return user, nil
}
