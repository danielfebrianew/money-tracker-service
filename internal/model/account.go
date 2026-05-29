package model

import "time"

type Account struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"`
	Balance   int       `json:"balance" db:"balance"`
	Icon      string    `json:"icon" db:"icon"`
	Color     string    `json:"color" db:"color"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CreateAccountInput struct {
	Name    string
	Type    string
	Balance int
	Icon    string
	Color   string
}

type UpdateAccountInput struct {
	Name  *string
	Icon  *string
	Color *string
}
