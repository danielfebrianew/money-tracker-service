package accounts

import "money-management-service/internal/model"

type CreateRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type UpdateRequest struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}

type CreateInput = model.CreateAccountInput
type UpdateInput = model.UpdateAccountInput
