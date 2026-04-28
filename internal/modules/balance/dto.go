package balance

import "time"

type Response struct {
	Balance       int        `json:"balance"`
	PlanType      string     `json:"plan_type"`
	ExpiresAt     *time.Time `json:"expires_at"`
	DaysRemaining int        `json:"days_remaining"`
	IsGracePeriod bool       `json:"is_grace_period"`
}
