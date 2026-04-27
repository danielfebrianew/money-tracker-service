package model

import "time"

type Admin struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         string    `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type AdminLog struct {
	ID         string    `json:"id" db:"id"`
	AdminID    string    `json:"admin_id" db:"admin_id"`
	Action     string    `json:"action" db:"action"`
	TargetType *string   `json:"target_type,omitempty" db:"target_type"`
	TargetID   *string   `json:"target_id,omitempty" db:"target_id"`
	Detail     *string   `json:"detail,omitempty" db:"detail"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type AdminLogView struct {
	AdminLog
	AdminUsername string `json:"admin_username" db:"admin_username"`
}
