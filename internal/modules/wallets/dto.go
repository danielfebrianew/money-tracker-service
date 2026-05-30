package wallets

import "money-tracker-service/internal/model"

type CreateRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Balance int    `json:"balance"`
	Icon    string `json:"icon"`
	Color   string `json:"color"`
}

type UpdateRequest struct {
	Name  *string `json:"name"`
	Icon  *string `json:"icon"`
	Color *string `json:"color"`
}

type CreateInput = model.CreateWalletInput
type UpdateInput = model.UpdateWalletInput
