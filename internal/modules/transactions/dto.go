package transactions

import "money-management-service/internal/model"

type CreateRequest struct {
	Deskripsi string `json:"deskripsi"`
	Jumlah    int    `json:"jumlah"`
	Kategori  string `json:"kategori"`
	Tipe      string `json:"tipe"`
}

type CreateInput = model.CreateTransactionInput
type Filters = model.TransactionFilters
type ListParams = model.TransactionListParams
