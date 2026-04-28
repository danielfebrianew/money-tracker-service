package service

import (
	balancemodule "money-management-service/internal/modules/balance"
	"money-management-service/internal/repository"
)

type BalanceService = balancemodule.Service

func NewBalanceService(store *repository.Store) *BalanceService {
	return balancemodule.NewService(balancemodule.NewRepository(store.DB()))
}
