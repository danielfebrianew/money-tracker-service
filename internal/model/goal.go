package model

import "time"

type Goal struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id"`
	Name          string    `json:"name" db:"name"`
	TargetAmount  int       `json:"target_amount" db:"target_amount"`
	CurrentAmount int       `json:"current_amount" db:"current_amount"`
	Deadline      string    `json:"deadline" db:"deadline"`
	Icon          string    `json:"icon" db:"icon"`
	Color         string    `json:"color" db:"color"`
	Status        string    `json:"status" db:"status"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
