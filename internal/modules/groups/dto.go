package groups

type CreateRequest struct {
	Name string `json:"name"`
}

type InviteRequest struct {
	Phone string `json:"phone"`
}

type TransactionRequest struct {
	Deskripsi string `json:"deskripsi"`
	Jumlah    int    `json:"jumlah"`
	Kategori  string `json:"kategori"`
	Tipe      string `json:"tipe"`
}
