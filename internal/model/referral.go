package model

import "time"

type ReferralCode struct {
	ID         string    `json:"id" db:"id"`
	UserID     *string   `json:"user_id,omitempty" db:"user_id"`
	Code       string    `json:"code" db:"code"`
	Name       string    `json:"name" db:"name"`
	Phone      *string   `json:"phone,omitempty" db:"phone"`
	Commission int       `json:"commission" db:"commission"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type ReferralSignup struct {
	ID           string    `json:"id" db:"id"`
	ReferralCode string    `json:"referral_code" db:"referral_code"`
	UserID       string    `json:"user_id" db:"user_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type ReferralPayout struct {
	ID           string     `json:"id" db:"id"`
	ReferralCode string     `json:"referral_code" db:"referral_code"`
	Amount       int        `json:"amount" db:"amount"`
	Period       string     `json:"period" db:"period"`
	Status       string     `json:"status" db:"status"`
	PaidAt       *time.Time `json:"paid_at,omitempty" db:"paid_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}
