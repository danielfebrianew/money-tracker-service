package model

import "time"

type BudgetGroup struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	OwnerID   string    `json:"owner_id" db:"owner_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type BudgetGroupMember struct {
	GroupID  string    `json:"group_id" db:"group_id"`
	UserID   string    `json:"user_id" db:"user_id"`
	Role     string    `json:"role" db:"role"`
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}

type GroupListItem struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Role        string `json:"role" db:"role"`
	MemberCount int    `json:"member_count" db:"member_count"`
}

type GroupMemberView struct {
	UserID string `json:"user_id" db:"user_id"`
	Name   string `json:"name" db:"name"`
	Phone  string `json:"phone,omitempty" db:"phone"`
	Role   string `json:"role" db:"role"`
}

type GroupMemberTotal struct {
	Phone   string  `json:"phone" db:"phone"`
	Name    string  `json:"name" db:"name"`
	Total   int     `json:"total" db:"total"`
	Percent float64 `json:"percent" db:"percent"`
}

type GroupReport struct {
	GroupName  string             `json:"group_name"`
	Month      string             `json:"month"`
	TotalOut   int                `json:"total_out"`
	ByMember   []GroupMemberTotal `json:"by_member"`
	ByKategori []CategoryTotal    `json:"by_kategori"`
}
