package budget

type CreateInput struct {
	Kategori string `json:"kategori"`
	Limit    int    `json:"limit"`
	Month    string `json:"month"`
}

type UpdateInput struct {
	Limit int `json:"limit"`
}
