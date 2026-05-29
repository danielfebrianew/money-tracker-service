package model

import "time"

type Budget struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Kategori  string    `json:"kategori" db:"kategori"`
	Limit     int       `json:"limit" db:"limit"`
	Month     string    `json:"month" db:"month"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type BudgetWithSpent struct {
	Budget
	Spent int `json:"spent" db:"spent"`
}

type BudgetHistory struct {
	Month      string `json:"month" db:"month"`
	Kategori   string `json:"kategori" db:"kategori"`
	TotalSpent int    `json:"total_spent" db:"total_spent"`
}
