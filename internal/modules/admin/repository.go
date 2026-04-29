package admin

import "money-management-service/internal/repository"

type Repository struct {
	store *repository.Store
}

func NewRepository(store *repository.Store) *Repository {
	return &Repository{store: store}
}
