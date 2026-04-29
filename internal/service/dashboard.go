package service

import (
	"money-management-service/internal/cache"
	dashboardmodule "money-management-service/internal/modules/dashboard"
	"money-management-service/internal/repository"
)

type DashboardService = dashboardmodule.Service

func NewDashboardService(cache *cache.Cache, store *repository.Store) *DashboardService {
	return dashboardmodule.NewService(cache, dashboardmodule.NewRepository(store.DB()))
}
