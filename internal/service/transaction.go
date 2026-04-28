package service

import (
	"money-management-service/internal/cache"
	transactions "money-management-service/internal/modules/transactions"
	"money-management-service/internal/repository"
)

type TransactionService = transactions.Service

func NewTransactionService(store *repository.Store, cache *cache.Cache, parser OpenAIService) *TransactionService {
	return transactions.NewService(transactions.NewRepository(store.DB()), cache, parser)
}
