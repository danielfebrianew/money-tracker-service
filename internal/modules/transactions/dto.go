package transactions

import "money-tracker-service/internal/model"

type CreateRequest struct {
	Deskripsi string  `json:"deskripsi"`
	Jumlah    int     `json:"jumlah"`
	Kategori  string  `json:"kategori"`
	Tipe      string  `json:"tipe"`
	WalletID *string `json:"wallet_id"`
}

type CreateInput = model.CreateTransactionInput
type Filters = model.TransactionFilters
type ListParams = model.TransactionListParams
