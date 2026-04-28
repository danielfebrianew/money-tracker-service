package service

import (
	tokensmodule "money-management-service/internal/modules/tokens"
	"money-management-service/internal/repository"
)

type TokenService = tokensmodule.Service

func NewTokenService(store *repository.Store) *TokenService {
	return tokensmodule.NewService(tokensmodule.NewRepository(store.DB()))
}
