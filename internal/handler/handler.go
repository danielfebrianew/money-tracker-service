package handler

import (
	authmodule "money-management-service/internal/modules/auth"
	balancemodule "money-management-service/internal/modules/balance"
	paymentsmodule "money-management-service/internal/modules/payments"
	tokensmodule "money-management-service/internal/modules/tokens"
	transactions "money-management-service/internal/modules/transactions"
	"money-management-service/internal/service"
)

type Handler struct {
	Health       *HealthHandler
	Auth         *authmodule.Module
	User         *UserHandler
	Balance      *balancemodule.Module
	Tokens       *tokensmodule.Module
	Payments     *paymentsmodule.Module
	Transactions *transactions.Module
	Dashboard    *DashboardHandler
	Groups       *GroupHandler
	Referral     *ReferralHandler
	Admin        *AdminHandler
	Webhook      *WebhookHandler
}

type Dependencies struct {
	Auth         *authmodule.Module
	User         *service.UserService
	Balance      *balancemodule.Module
	Tokens       *tokensmodule.Module
	Payments     *paymentsmodule.Module
	Transactions *transactions.Module
	Dashboard    *service.DashboardService
	Groups       *service.GroupService
	Referral     *service.ReferralService
	Admin        *service.AdminService
	Webhook      *service.WebhookService
}

func New(deps Dependencies) *Handler {
	return &Handler{
		Health:       NewHealthHandler(),
		Auth:         deps.Auth,
		User:         NewUserHandler(deps.User),
		Balance:      deps.Balance,
		Tokens:       deps.Tokens,
		Payments:     deps.Payments,
		Transactions: deps.Transactions,
		Dashboard:    NewDashboardHandler(deps.Dashboard),
		Groups:       NewGroupHandler(deps.Groups, deps.Transactions.Service),
		Referral:     NewReferralHandler(deps.Referral),
		Admin:        NewAdminHandler(deps.Auth.Service, deps.Admin, deps.Payments.Service),
		Webhook:      NewWebhookHandler(deps.Webhook),
	}
}
