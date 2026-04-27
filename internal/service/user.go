package service

import (
	"context"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	"money-management-service/internal/repository"
)

type UserService struct {
	store *repository.Store
	cache *cache.Cache
}

func NewUserService(store *repository.Store, cache *cache.Cache) *UserService {
	return &UserService{store: store, cache: cache}
}

func (s *UserService) Profile(ctx context.Context, userID string) (*model.User, *model.UserBalance, error) {
	var cached struct {
		User    *model.User        `json:"user"`
		Balance *model.UserBalance `json:"balance"`
	}
	if s.cache.GetJSON(ctx, "user:"+userID, &cached) && cached.User != nil && cached.Balance != nil {
		return cached.User, cached.Balance, nil
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	balance, err := s.store.GetBalance(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	s.cache.SetJSON(ctx, "user:"+userID, map[string]interface{}{"user": user, "balance": balance}, 15*time.Minute)
	return user, balance, nil
}

func (s *UserService) Update(ctx context.Context, userID string, name, email, timezone *string) (*model.User, error) {
	user, err := s.store.UpdateUser(ctx, userID, name, email, timezone)
	if err != nil {
		return nil, err
	}
	s.cache.Delete(ctx, "user:"+userID)
	return user, nil
}
