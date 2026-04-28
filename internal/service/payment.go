package service

import (
	"time"

	"money-management-service/internal/cache"
	paymentsmodule "money-management-service/internal/modules/payments"
	"money-management-service/internal/repository"
)

type PaymentService = paymentsmodule.Service

func NewPaymentService(store *repository.Store, cache *cache.Cache) *PaymentService {
	return paymentsmodule.NewService(paymentsmodule.NewRepository(store.DB()), cache)
}

func CalculateExpiresAt(amount int) *time.Time {
	return paymentsmodule.CalculateExpiresAt(amount)
}
