package service

import (
	"money-management-service/internal/cache"
	"money-management-service/internal/config"
	authmodule "money-management-service/internal/modules/auth"
	"money-management-service/internal/repository"
)

type AuthService = authmodule.Service
type TokenPair = authmodule.TokenPair
type AppClaims = authmodule.AppClaims

func NewAuthService(cfg config.Config, store *repository.Store, cache *cache.Cache) *AuthService {
	return authmodule.NewService(cfg, authmodule.NewRepository(store.DB()), cache)
}

func HashToken(token string) string {
	return authmodule.HashToken(token)
}

func IsAppError(err error, target error) bool {
	return authmodule.IsAppError(err, target)
}
