package model

import "time"

type Payment struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"user_id" db:"user_id"`
	Type        string     `json:"type" db:"type"`
	Amount      int        `json:"amount" db:"amount"`
	Description *string    `json:"description,omitempty" db:"description"`
	ProofURL    *string    `json:"proof_url,omitempty" db:"proof_url"`
	Status      string     `json:"status" db:"status"`
	VerifiedBy  *string    `json:"verified_by,omitempty" db:"verified_by"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

type PaymentWithUser struct {
	Payment
	UserName  string `json:"user_name" db:"user_name"`
	UserPhone string `json:"user_phone" db:"user_phone"`
}
