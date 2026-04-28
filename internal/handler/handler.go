package handler

import "money-management-service/internal/service"

type Handler struct {
	Health       *HealthHandler
	Auth         *AuthHandler
	User         *UserHandler
	Balance      *BalanceHandler
	Tokens       *TokenHandler
	Payments     *PaymentHandler
	Transactions *TransactionHandler
	Dashboard    *DashboardHandler
	Groups       *GroupHandler
	Referral     *ReferralHandler
	Admin        *AdminHandler
	Webhook      *WebhookHandler
}

type Dependencies struct {
	Auth         *service.AuthService
	User         *service.UserService
	Balance      *service.BalanceService
	Tokens       *service.TokenService
	Payments     *service.PaymentService
	Transactions *service.TransactionService
	Dashboard    *service.DashboardService
	Groups       *service.GroupService
	Referral     *service.ReferralService
	Admin        *service.AdminService
	Webhook      *service.WebhookService
}

func New(deps Dependencies) *Handler {
	return &Handler{
		Health:       NewHealthHandler(),
		Auth:         NewAuthHandler(deps.Auth),
		User:         NewUserHandler(deps.User),
		Balance:      NewBalanceHandler(deps.Balance),
		Tokens:       NewTokenHandler(deps.Tokens),
		Payments:     NewPaymentHandler(deps.Payments),
		Transactions: NewTransactionHandler(deps.Transactions),
		Dashboard:    NewDashboardHandler(deps.Dashboard),
		Groups:       NewGroupHandler(deps.Groups, deps.Transactions),
		Referral:     NewReferralHandler(deps.Referral),
		Admin:        NewAdminHandler(deps.Auth, deps.Admin, deps.Payments),
		Webhook:      NewWebhookHandler(deps.Webhook),
	}
}
