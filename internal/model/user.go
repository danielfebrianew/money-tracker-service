package model

import "time"

type User struct {
	ID           string    `json:"id" db:"id"`
	Phone        string    `json:"phone" db:"phone"`
	Email        *string   `json:"email,omitempty" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	Timezone     string    `json:"timezone" db:"timezone"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type UserBalance struct {
	UserID    string     `json:"user_id" db:"user_id"`
	Balance   int        `json:"balance" db:"balance"`
	PlanType  string     `json:"plan_type" db:"plan_type"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

type UserWithBalance struct {
	User
	Balance   int        `json:"balance" db:"balance"`
	PlanType  string     `json:"plan_type" db:"plan_type"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}
