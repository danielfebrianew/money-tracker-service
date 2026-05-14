package model

import "time"

type Account struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"`
	Balance   int       `json:"balance" db:"balance"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CreateAccountInput struct {
	Name string
	Type string
}

type UpdateAccountInput struct {
	Name *string
	Type *string
}
