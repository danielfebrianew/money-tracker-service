package admin

import (
	"context"
	"fmt"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	paymentsmodule "money-management-service/internal/modules/payments"
	"money-management-service/internal/pkg/ids"
)

type Service struct {
	repository *Repository
	cache      *cache.Cache
}

func NewService(repository *Repository, cache *cache.Cache) *Service {
	return &Service{repository: repository, cache: cache}
}

func (s *Service) Dashboard(ctx context.Context) (map[string]interface{}, error) {
	var cached map[string]interface{}
	if s.cache.GetJSON(ctx, "admin:stats", &cached) {
		return cached, nil
	}
	stats, err := s.repository.DashboardStats(ctx)
	if err != nil {
		return nil, err
	}
	revenue := intFromMap(stats, "revenue_this_month")
	prevRevenue := intFromMap(stats, "revenue_prev_month")
	infraCost := 478600
	referralCost := 125000
	netProfit := revenue - infraCost - referralCost
	stats["infra_cost_this_month"] = infraCost
	stats["referral_cost_this_month"] = referralCost
	stats["net_profit"] = netProfit
	stats["profit_split"] = map[string]int{
		"daniel_75": netProfit * 75 / 100,
		"teman_25":  netProfit * 25 / 100,
	}
	change := 0.0
	if prevRevenue > 0 {
		change = round1(float64(revenue-prevRevenue) / float64(prevRevenue) * 100)
	}
	stats["revenue_change_percent"] = change
	s.cache.SetJSON(ctx, "admin:stats", stats, 2*time.Minute)
	return stats, nil
}

func (s *Service) ListUsers(ctx context.Context, status, search, sort, order string, page, perPage int) ([]model.UserWithBalance, int64, error) {
	return s.repository.ListUsers(ctx, status, search, sort, order, page, perPage)
}

func (s *Service) UserDetail(ctx context.Context, userID string) (map[string]interface{}, error) {
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	balance, err := s.repository.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	payments, _, err := s.repository.ListUserPayments(ctx, userID, "", 1, 10)
	if err != nil {
		return nil, err
	}
	stats := map[string]interface{}{
		"total_transactions":           s.repository.CountTransactions(ctx, userID),
		"total_wa_messages_this_month": 0,
		"total_ai_calls_this_month":    0,
		"ai_cost_this_month":           0,
		"registered_via_referral":      nil,
		"member_since_days":            int(time.Since(user.CreatedAt).Hours() / 24),
	}
	return map[string]interface{}{"user": user, "balance": balance, "payments": payments, "stats": stats}, nil
}

func (s *Service) UpdateUserStatus(ctx context.Context, adminID, userID string, active bool, reason string) error {
	if err := s.repository.UpdateUserStatus(ctx, userID, active); err != nil {
		return err
	}
	detail := fmt.Sprintf("Set active=%v. %s", active, reason)
	_ = s.Log(ctx, adminID, "update_user_status", strPtr("user"), &userID, &detail)
	s.cache.Delete(ctx, "user:"+userID, "admin:stats")
	return nil
}

func (s *Service) AddUserBalance(ctx context.Context, adminID, userID string, amount int, description string) (*model.UserBalance, error) {
	balance, err := s.repository.AddBalance(ctx, userID, amount, paymentsmodule.CalculateExpiresAt(amount))
	if err != nil {
		return nil, err
	}
	payment := model.Payment{
		ID:          ids.New("pay"),
		UserID:      userID,
		Type:        "topup",
		Amount:      amount,
		Description: &description,
		Status:      "verified",
		VerifiedBy:  &adminID,
		CreatedAt:   time.Now().UTC(),
	}
	now := time.Now().UTC()
	payment.VerifiedAt = &now
	_ = s.repository.CreatePayment(ctx, payment)
	detail := fmt.Sprintf("Added balance Rp%d for user %s", amount, userID)
	_ = s.Log(ctx, adminID, "add_balance", strPtr("user"), &userID, &detail)
	s.cache.Delete(ctx, "user:"+userID, "admin:stats")
	return balance, nil
}

func (s *Service) Revenue(ctx context.Context, month string) (map[string]interface{}, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	key := "admin:revenue:" + month
	var cached map[string]interface{}
	if s.cache.GetJSON(ctx, key, &cached) {
		return cached, nil
	}
	revenue, err := s.repository.Revenue(ctx, month)
	if err != nil {
		return nil, err
	}
	cost := map[string]int{"vps": 204000, "fonnte": 66000, "domain": 4600, "openai": 350000}
	totalInfra := cost["vps"] + cost["fonnte"] + cost["domain"] + cost["openai"]
	cost["total_infra"] = totalInfra
	referralPayout := 125000
	netProfit := revenue - totalInfra - referralPayout
	result := map[string]interface{}{
		"month":           month,
		"revenue":         revenue,
		"cost":            cost,
		"referral_payout": referralPayout,
		"net_profit":      netProfit,
		"profit_split": map[string]int{
			"daniel_75": netProfit * 75 / 100,
			"teman_25":  netProfit * 25 / 100,
		},
		"trend": []map[string]interface{}{},
	}
	s.cache.SetJSON(ctx, key, result, 5*time.Minute)
	return result, nil
}

func (s *Service) ReferralOverview(ctx context.Context) (map[string]interface{}, error) {
	return s.repository.ReferralOverview(ctx)
}

func (s *Service) CreateReferralPayout(ctx context.Context, adminID, referralCode, period string) (map[string]interface{}, error) {
	data, err := s.repository.CreateReferralPayout(ctx, referralCode, period)
	if err != nil {
		return nil, err
	}
	detail := fmt.Sprintf("Referral payout %s period %s amount %v", referralCode, period, data["amount"])
	_ = s.Log(ctx, adminID, "referral_payout", strPtr("referral"), &referralCode, &detail)
	return data, nil
}

func (s *Service) Logs(ctx context.Context, adminID, action string, page, perPage int) ([]model.AdminLogView, int64, error) {
	return s.repository.ListLogs(ctx, adminID, action, page, perPage)
}

func (s *Service) Log(ctx context.Context, adminID, action string, targetType, targetID, detail *string) error {
	return s.repository.CreateLog(ctx, model.AdminLog{
		ID:         ids.New("log"),
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		CreatedAt:  time.Now().UTC(),
	})
}
