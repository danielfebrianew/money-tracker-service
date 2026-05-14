package handler

import (
	accountsmodule "money-management-service/internal/modules/accounts"
	adminmodule "money-management-service/internal/modules/admin"
	authmodule "money-management-service/internal/modules/auth"
	balancemodule "money-management-service/internal/modules/balance"
	dashboardmodule "money-management-service/internal/modules/dashboard"
	groupsmodule "money-management-service/internal/modules/groups"
	paymentsmodule "money-management-service/internal/modules/payments"
	referralmodule "money-management-service/internal/modules/referral"
	tokensmodule "money-management-service/internal/modules/tokens"
	transactions "money-management-service/internal/modules/transactions"
	usersmodule "money-management-service/internal/modules/users"
	webhookmodule "money-management-service/internal/modules/webhook"
)

type Handler struct {
	Health       *HealthHandler
	Auth         *authmodule.Module
	User         *usersmodule.Module
	Balance      *balancemodule.Module
	Tokens       *tokensmodule.Module
	Payments     *paymentsmodule.Module
	Transactions *transactions.Module
	Accounts     *accountsmodule.Module
	Dashboard    *dashboardmodule.Module
	Groups       *groupsmodule.Module
	Referral     *referralmodule.Module
	Admin        *adminmodule.Module
	Webhook      *webhookmodule.Module
}

type Dependencies struct {
	Auth         *authmodule.Module
	User         *usersmodule.Module
	Balance      *balancemodule.Module
	Tokens       *tokensmodule.Module
	Payments     *paymentsmodule.Module
	Transactions *transactions.Module
	Accounts     *accountsmodule.Module
	Dashboard    *dashboardmodule.Module
	Groups       *groupsmodule.Module
	Referral     *referralmodule.Module
	Admin        *adminmodule.Module
	Webhook      *webhookmodule.Module
}

func New(deps Dependencies) *Handler {
	return &Handler{
		Health:       NewHealthHandler(),
		Auth:         deps.Auth,
		User:         deps.User,
		Balance:      deps.Balance,
		Tokens:       deps.Tokens,
		Payments:     deps.Payments,
		Transactions: deps.Transactions,
		Accounts:     deps.Accounts,
		Dashboard:    deps.Dashboard,
		Groups:       deps.Groups,
		Referral:     deps.Referral,
		Admin:        deps.Admin,
		Webhook:      deps.Webhook,
	}
}
