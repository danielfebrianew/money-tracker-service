package auth

type RegisterRequest struct {
	Phone        string  `json:"phone"`
	Name         string  `json:"name"`
	Email        *string `json:"email"`
	Password     string  `json:"password"`
	ReferralCode *string `json:"referral_code"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"` // email (default) atau nomor telepon (628xxx)
	Password   string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
