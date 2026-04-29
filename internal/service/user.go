package service

import (
	"money-management-service/internal/cache"
	usersmodule "money-management-service/internal/modules/users"
	"money-management-service/internal/repository"
)

type UserService = usersmodule.Service

func NewUserService(store *repository.Store, cache *cache.Cache) *UserService {
	return usersmodule.NewService(usersmodule.NewRepository(store.DB()), cache)
}
