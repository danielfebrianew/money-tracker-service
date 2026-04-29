package service

import (
	"money-management-service/internal/cache"
	groupsmodule "money-management-service/internal/modules/groups"
	"money-management-service/internal/repository"
)

type GroupService = groupsmodule.Service

func NewGroupService(store *repository.Store, cache *cache.Cache, transactions *TransactionService) *GroupService {
	return groupsmodule.NewService(groupsmodule.NewRepository(store.DB()), cache, transactions)
}
