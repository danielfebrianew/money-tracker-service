package categories

type CreateInput struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

type UpdateInput struct {
	Label *string `json:"label"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}
