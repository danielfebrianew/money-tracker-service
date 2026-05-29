package handler

import (
	accountsmodule "money-tracker-service/internal/modules/accounts"
	adminmodule "money-tracker-service/internal/modules/admin"
	authmodule "money-tracker-service/internal/modules/auth"
	balancemodule "money-tracker-service/internal/modules/balance"
	dashboardmodule "money-tracker-service/internal/modules/dashboard"
	groupsmodule "money-tracker-service/internal/modules/groups"
	paymentsmodule "money-tracker-service/internal/modules/payments"
	referralmodule "money-tracker-service/internal/modules/referral"
	tokensmodule "money-tracker-service/internal/modules/tokens"
	transactions "money-tracker-service/internal/modules/transactions"
	usersmodule "money-tracker-service/internal/modules/users"
	webhookmodule "money-tracker-service/internal/modules/webhook"
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
