package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/ids"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DashboardStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{}
	queries := map[string]string{
		"total_users":          `SELECT COUNT(*) FROM users`,
		"active_users":         `SELECT COUNT(*) FROM users WHERE is_active = TRUE`,
		"suspended_users":      `SELECT COUNT(*) FROM users WHERE is_active = FALSE`,
		"pending_payments":     `SELECT COUNT(*) FROM payments WHERE status = 'pending'`,
		"revenue_this_month":   `SELECT COALESCE(SUM(amount),0) FROM payments WHERE status = 'verified' AND created_at >= date_trunc('month', NOW())`,
		"revenue_prev_month":   `SELECT COALESCE(SUM(amount),0) FROM payments WHERE status = 'verified' AND created_at >= date_trunc('month', NOW()) - INTERVAL '1 month' AND created_at < date_trunc('month', NOW())`,
		"new_users_this_month": `SELECT COUNT(*) FROM users WHERE created_at >= date_trunc('month', NOW())`,
		"churn_this_month":     `SELECT COUNT(*) FROM users WHERE is_active = FALSE AND updated_at >= date_trunc('month', NOW())`,
	}
	for key, query := range queries {
		var value int
		if err := r.db.GetContext(ctx, &value, query); err != nil {
			return nil, err
		}
		stats[key] = value
	}
	return stats, nil
}

func (r *Repository) ListUsers(ctx context.Context, status, search, sort, order string, page, perPage int) ([]model.UserWithBalance, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	if status == "active" {
		where = append(where, "u.is_active = TRUE")
	}
	if status == "inactive" || status == "suspended" {
		where = append(where, "u.is_active = FALSE")
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		where = append(where, fmt.Sprintf("(u.name ILIKE $%d OR u.phone ILIKE $%d OR COALESCE(u.email, '') ILIKE $%d)", len(args), len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM users u WHERE `+whereSQL, args...); err != nil {
		return nil, 0, err
	}

	sortColumn := "u.created_at"
	switch sort {
	case "name":
		sortColumn = "u.name"
	case "phone":
		sortColumn = "u.phone"
	case "balance":
		sortColumn = "b.balance"
	}
	if strings.ToLower(order) != "asc" {
		order = "desc"
	} else {
		order = "asc"
	}
	args = append(args, perPage, offset(page, perPage))
	var users []model.UserWithBalance
	err := r.db.SelectContext(ctx, &users, `
		SELECT u.*, b.balance, b.plan_type, b.expires_at
		FROM users u
		JOIN user_balances b ON b.user_id = u.id
		WHERE `+whereSQL+`
		ORDER BY `+sortColumn+` `+order+`
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return users, total, err
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &user, err
}

func (r *Repository) GetBalance(ctx context.Context, userID string) (*model.UserBalance, error) {
	var balance model.UserBalance
	err := r.db.GetContext(ctx, &balance, `SELECT * FROM user_balances WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &balance, err
}

func (r *Repository) ListUserPayments(ctx context.Context, userID, status string, page, perPage int) ([]model.Payment, int64, error) {
	where := []string{"user_id = $1"}
	args := []interface{}{userID}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM payments WHERE `+whereSQL, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, perPage, offset(page, perPage))
	var items []model.Payment
	err := r.db.SelectContext(ctx, &items, `
		SELECT * FROM payments
		WHERE `+whereSQL+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return items, total, err
}

func (r *Repository) CountTransactions(ctx context.Context, userID string) int {
	var total int
	_ = r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM transactions WHERE user_id = $1`, userID)
	return total
}

func (r *Repository) UpdateUserStatus(ctx context.Context, userID string, active bool) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET is_active = $2, updated_at = NOW() WHERE id = $1
	`, userID, active)
	return execOrNotFound(res, err)
}

func (r *Repository) AddBalance(ctx context.Context, userID string, amount int, expiresAt *time.Time) (*model.UserBalance, error) {
	var balance model.UserBalance
	err := r.db.GetContext(ctx, &balance, `
		UPDATE user_balances
		SET balance = balance + $2,
			expires_at = COALESCE($3, expires_at),
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING *
	`, userID, amount, expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &balance, err
}

func (r *Repository) CreatePayment(ctx context.Context, payment model.Payment) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO payments (id, user_id, type, amount, description, proof_url, status, verified_by, verified_at, created_at)
		VALUES (:id, :user_id, :type, :amount, :description, :proof_url, :status, :verified_by, :verified_at, :created_at)
	`, payment)
	return err
}

func (r *Repository) Revenue(ctx context.Context, month string) (int, error) {
	var revenue int
	err := r.db.GetContext(ctx, &revenue, `
		SELECT COALESCE(SUM(amount),0)
		FROM payments
		WHERE status = 'verified' AND to_char(created_at, 'YYYY-MM') = $1
	`, month)
	return revenue, err
}

func (r *Repository) ReferralOverview(ctx context.Context) (map[string]interface{}, error) {
	rows, err := r.db.QueryxContext(ctx, `
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

func (r *Repository) CreateReferralPayout(ctx context.Context, referralCode, period string) (map[string]interface{}, error) {
	var amount int
	err := r.db.GetContext(ctx, &amount, `
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
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO referral_payouts (id, referral_code, amount, period, status, paid_at, created_at)
		VALUES ($1, $2, $3, $4, 'paid', NOW(), NOW())
	`, ids.New("pout"), referralCode, amount, period)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"referral_code": referralCode, "amount": amount, "period": period}, nil
}

func (r *Repository) ListLogs(ctx context.Context, adminID, action string, page, perPage int) ([]model.AdminLogView, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	if adminID != "" {
		args = append(args, adminID)
		where = append(where, fmt.Sprintf("l.admin_id = $%d", len(args)))
	}
	if action != "" {
		args = append(args, action)
		where = append(where, fmt.Sprintf("l.action = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM admin_logs l WHERE `+whereSQL, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, perPage, offset(page, perPage))
	var logs []model.AdminLogView
	err := r.db.SelectContext(ctx, &logs, `
		SELECT l.*, a.username AS admin_username
		FROM admin_logs l
		JOIN admins a ON a.id = l.admin_id
		WHERE `+whereSQL+`
		ORDER BY l.created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return logs, total, err
}

func (r *Repository) CreateLog(ctx context.Context, log model.AdminLog) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO admin_logs (id, admin_id, action, target_type, target_id, detail, created_at)
		VALUES (:id, :admin_id, :action, :target_type, :target_id, :detail, :created_at)
	`, log)
	return err
}

func intFromMap(values map[string]interface{}, key string) int {
	value, _ := values[key].(int)
	return value
}

func round1(value float64) float64 {
	return float64(int(value*10)) / 10
}

func offset(page, perPage int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * perPage
}

func execOrNotFound(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
