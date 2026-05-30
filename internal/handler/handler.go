package handler

import (
	walletsmodule "money-tracker-service/internal/modules/wallets"
	adminmodule "money-tracker-service/internal/modules/admin"
	goalsmodule "money-tracker-service/internal/modules/goals"
	authmodule "money-tracker-service/internal/modules/auth"
	balancemodule "money-tracker-service/internal/modules/balance"
	budgetmodule "money-tracker-service/internal/modules/budget"
	categoriesmodule "money-tracker-service/internal/modules/categories"
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
	Wallets      *walletsmodule.Module
	Dashboard    *dashboardmodule.Module
	Groups       *groupsmodule.Module
	Referral     *referralmodule.Module
	Admin        *adminmodule.Module
	Webhook      *webhookmodule.Module
	Budget       *budgetmodule.Module
	Categories   *categoriesmodule.Module
	Goals        *goalsmodule.Module
}

type Dependencies struct {
	Auth         *authmodule.Module
	User         *usersmodule.Module
	Balance      *balancemodule.Module
	Tokens       *tokensmodule.Module
	Payments     *paymentsmodule.Module
	Transactions *transactions.Module
	Wallets      *walletsmodule.Module
	Dashboard    *dashboardmodule.Module
	Groups       *groupsmodule.Module
	Referral     *referralmodule.Module
	Admin        *adminmodule.Module
	Webhook      *webhookmodule.Module
	Budget       *budgetmodule.Module
	Categories   *categoriesmodule.Module
	Goals        *goalsmodule.Module
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
		Wallets:      deps.Wallets,
		Dashboard:    deps.Dashboard,
		Groups:       deps.Groups,
		Referral:     deps.Referral,
		Admin:        deps.Admin,
		Webhook:      deps.Webhook,
		Budget:       deps.Budget,
		Categories:   deps.Categories,
		Goals:        deps.Goals,
	}
}
