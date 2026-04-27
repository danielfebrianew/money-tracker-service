package service

import (
	"context"
	"fmt"
	"time"

	"money-management-service/internal/cache"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
)

type AdminService struct {
	store *repository.Store
	cache *cache.Cache
}

func NewAdminService(store *repository.Store, cache *cache.Cache) *AdminService {
	return &AdminService{store: store, cache: cache}
}

func (s *AdminService) Dashboard(ctx context.Context) (map[string]interface{}, error) {
	var cached map[string]interface{}
	if s.cache.GetJSON(ctx, "admin:stats", &cached) {
		return cached, nil
	}
	stats, err := s.store.AdminDashboardStats(ctx)
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
		change = repository.Round1(float64(revenue-prevRevenue) / float64(prevRevenue) * 100)
	}
	stats["revenue_change_percent"] = change
	s.cache.SetJSON(ctx, "admin:stats", stats, 2*time.Minute)
	return stats, nil
}

func (s *AdminService) ListUsers(ctx context.Context, status, search, sort, order string, page, perPage int) ([]model.UserWithBalance, int64, error) {
	return s.store.ListAdminUsers(ctx, status, search, sort, order, page, perPage)
}

func (s *AdminService) UserDetail(ctx context.Context, userID string) (map[string]interface{}, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	balance, err := s.store.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	payments, _, err := s.store.ListUserPayments(ctx, userID, "", 1, 10)
	if err != nil {
		return nil, err
	}
	var totalTransactions int
	_ = s.store.DB().GetContext(ctx, &totalTransactions, `SELECT COUNT(*) FROM transactions WHERE user_id = $1`, userID)
	stats := map[string]interface{}{
		"total_transactions":           totalTransactions,
		"total_wa_messages_this_month": 0,
		"total_ai_calls_this_month":    0,
		"ai_cost_this_month":           0,
		"registered_via_referral":      nil,
		"member_since_days":            int(time.Since(user.CreatedAt).Hours() / 24),
	}
	return map[string]interface{}{"user": user, "balance": balance, "payments": payments, "stats": stats}, nil
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, adminID, userID string, active bool, reason string) error {
	if err := s.store.UpdateUserStatus(ctx, userID, active); err != nil {
		return err
	}
	detail := fmt.Sprintf("Set active=%v. %s", active, reason)
	_ = s.Log(ctx, adminID, "update_user_status", strPtr("user"), &userID, &detail)
	s.cache.Delete(ctx, "user:"+userID, "admin:stats")
	return nil
}

func (s *AdminService) AddUserBalance(ctx context.Context, adminID, userID string, amount int, description string) (*model.UserBalance, error) {
	balance, err := s.store.AddBalance(ctx, userID, amount, CalculateExpiresAt(amount))
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
	_ = s.store.CreatePayment(ctx, payment)
	detail := fmt.Sprintf("Added balance Rp%d for user %s", amount, userID)
	_ = s.Log(ctx, adminID, "add_balance", strPtr("user"), &userID, &detail)
	s.cache.Delete(ctx, "user:"+userID, "admin:stats")
	return balance, nil
}

func (s *AdminService) Revenue(ctx context.Context, month string) (map[string]interface{}, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	key := "admin:revenue:" + month
	var cached map[string]interface{}
	if s.cache.GetJSON(ctx, key, &cached) {
		return cached, nil
	}
	var revenue int
	err := s.store.DB().GetContext(ctx, &revenue, `
		SELECT COALESCE(SUM(amount),0)
		FROM payments
		WHERE status = 'verified' AND to_char(created_at, 'YYYY-MM') = $1
	`, month)
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

func (s *AdminService) ReferralOverview(ctx context.Context) (map[string]interface{}, error) {
	rows, err := s.store.DB().QueryxContext(ctx, `
		SELECT
			rc.code,
			rc.name,
			COALESCE(rc.phone, '') AS phone,
			COUNT(rs.user_id) AS total_users,
			COUNT(rs.user_id) FILTER (WHERE u.is_active = TRUE) AS active_users,
			COALESCE(SUM(rp.amount) FILTER (WHERE rp.status = 'paid'), 0) AS earned,
			COUNT(rs.user_id) FILTER (WHERE u.is_active = TRUE) * MAX(rc.commission) AS pending
		FROM referral_codes rc
		LEFT JOIN referral_signups rs ON rs.referral_code = rc.code
		LEFT JOIN users u ON u.id = rs.user_id
		LEFT JOIN referral_payouts rp ON rp.referral_code = rc.code
		GROUP BY rc.code, rc.name, rc.phone
		ORDER BY total_users DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	referrers := []map[string]interface{}{}
	totalReferred := 0
	totalPaid := 0
	totalPending := 0
	for rows.Next() {
		var code, name, phone string
		var totalUsers, activeUsers, earned, pending int
		if err := rows.Scan(&code, &name, &phone, &totalUsers, &activeUsers, &earned, &pending); err != nil {
			return nil, err
		}
		totalReferred += totalUsers
		totalPaid += earned
		totalPending += pending
		referrers = append(referrers, map[string]interface{}{
			"code": code, "name": name, "phone": phone, "total_users": totalUsers,
			"active_users": activeUsers, "earned": earned, "pending": pending,
		})
	}
	return map[string]interface{}{
		"total_referrers":      len(referrers),
		"total_referred_users": totalReferred,
		"total_paid":           totalPaid,
		"total_pending":        totalPending,
		"referrers":            referrers,
	}, rows.Err()
}

func (s *AdminService) CreateReferralPayout(ctx context.Context, adminID, referralCode, period string) (map[string]interface{}, error) {
	var amount int
	err := s.store.DB().GetContext(ctx, &amount, `
		SELECT COUNT(rs.user_id) FILTER (WHERE u.is_active = TRUE) * MAX(rc.commission)
		FROM referral_codes rc
		LEFT JOIN referral_signups rs ON rs.referral_code = rc.code
		LEFT JOIN users u ON u.id = rs.user_id
		WHERE rc.code = $1
		GROUP BY rc.code
	`, referralCode)
	if err != nil {
		return nil, err
	}
	_, err = s.store.DB().ExecContext(ctx, `
		INSERT INTO referral_payouts (id, referral_code, amount, period, status, paid_at, created_at)
		VALUES ($1, $2, $3, $4, 'paid', NOW(), NOW())
	`, ids.New("pout"), referralCode, amount, period)
	if err != nil {
		return nil, err
	}
	detail := fmt.Sprintf("Referral payout %s period %s amount %d", referralCode, period, amount)
	_ = s.Log(ctx, adminID, "referral_payout", strPtr("referral"), &referralCode, &detail)
	return map[string]interface{}{"referral_code": referralCode, "amount": amount, "period": period}, nil
}

func (s *AdminService) Logs(ctx context.Context, adminID, action string, page, perPage int) ([]model.AdminLogView, int64, error) {
	return s.store.ListAdminLogs(ctx, adminID, action, page, perPage)
}

func (s *AdminService) Log(ctx context.Context, adminID, action string, targetType, targetID, detail *string) error {
	return s.store.CreateAdminLog(ctx, model.AdminLog{
		ID:         ids.New("log"),
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		CreatedAt:  time.Now().UTC(),
	})
}

func intFromMap(values map[string]interface{}, key string) int {
	value, _ := values[key].(int)
	return value
}

func strPtr(value string) *string {
	return &value
}
