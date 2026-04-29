package admin

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UpdateUserStatusRequest struct {
	IsActive bool   `json:"is_active"`
	Reason   string `json:"reason"`
}

type AddUserBalanceRequest struct {
	Amount      int    `json:"amount"`
	Description string `json:"description"`
}

type RejectPaymentRequest struct {
	Reason string `json:"reason"`
}

type ReferralPayoutRequest struct {
	ReferralCode string `json:"referral_code"`
	Period       string `json:"period"`
}
