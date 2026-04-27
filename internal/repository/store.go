package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sqlx.DB {
	return s.db
}

func (s *Store) CreateUserWithBalance(ctx context.Context, user *model.User, referralCode *string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO users (id, phone, email, password_hash, name, timezone, is_active, created_at, updated_at)
		VALUES (:id, :phone, :email, :password_hash, :name, :timezone, :is_active, :created_at, :updated_at)
	`, user)
	if err != nil {
		if isDuplicate(err) {
			return apperror.New(apperror.ErrConflict, "Nomor telepon sudah terdaftar")
		}
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_balances (user_id, balance, plan_type, updated_at)
		VALUES ($1, 0, 'topup', NOW())
	`, user.ID)
	if err != nil {
		return err
	}

	if referralCode != nil && strings.TrimSpace(*referralCode) != "" {
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO referral_signups (id, referral_code, user_id)
			SELECT $1, code, $3
			FROM referral_codes
			WHERE code = $2 AND is_active = TRUE
			ON CONFLICT (user_id) DO NOTHING
		`, ids.New("refsgn"), strings.ToUpper(strings.TrimSpace(*referralCode)), user.ID)
	}

	return tx.Commit()
}

func (s *Store) GetUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := s.db.GetContext(ctx, &user, `SELECT * FROM users WHERE phone = $1`, phone)
	return userOrNotFound(&user, err)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := s.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, id)
	return userOrNotFound(&user, err)
}

func (s *Store) UpdateUser(ctx context.Context, userID string, name, email, timezone *string) (*model.User, error) {
	var user model.User
	err := s.db.GetContext(ctx, &user, `
		UPDATE users
		SET
			name = COALESCE($2, name),
			email = COALESCE($3, email),
			timezone = COALESCE($4, timezone),
			updated_at = NOW()
		WHERE id = $1
		RETURNING *
	`, userID, name, email, timezone)
	return userOrNotFound(&user, err)
}

func (s *Store) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, userID, passwordHash)
	return execOrNotFound(res, err)
}

func (s *Store) UpdateUserStatus(ctx context.Context, userID string, active bool) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE users SET is_active = $2, updated_at = NOW() WHERE id = $1
	`, userID, active)
	return execOrNotFound(res, err)
}

func (s *Store) GetBalance(ctx context.Context, userID string) (*model.UserBalance, error) {
	var balance model.UserBalance
	err := s.db.GetContext(ctx, &balance, `SELECT * FROM user_balances WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &balance, err
}

func (s *Store) AddBalance(ctx context.Context, userID string, amount int, expiresAt *time.Time) (*model.UserBalance, error) {
	var balance model.UserBalance
	err := s.db.GetContext(ctx, &balance, `
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

func (s *Store) CreateRefreshToken(ctx context.Context, token model.RefreshToken) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES (:id, :user_id, :token_hash, :expires_at, :created_at)
	`, token)
	return err
}

func (s *Store) GetRefreshTokenByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := s.db.GetContext(ctx, &token, `SELECT * FROM refresh_tokens WHERE token_hash = $1`, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrUnauthorized
	}
	return &token, err
}

func (s *Store) DeleteRefreshTokenByHash(ctx context.Context, hash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, hash)
	return err
}

func (s *Store) PruneRefreshTokens(ctx context.Context, userID string, keep int) error {
	if keep < 0 {
		keep = 0
	}
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM refresh_tokens
		WHERE user_id = $1
		  AND id NOT IN (
			SELECT id FROM refresh_tokens
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		  )
	`, userID, keep)
	return err
}

func (s *Store) CreateAPIToken(ctx context.Context, token model.APIToken) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO api_tokens (id, user_id, token, name, last_used_at, created_at)
		VALUES (:id, :user_id, :token, :name, :last_used_at, :created_at)
	`, token)
	return err
}

func (s *Store) CountAPITokens(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM api_tokens WHERE user_id = $1`, userID)
	return count, err
}

func (s *Store) ListAPITokens(ctx context.Context, userID string) ([]model.APIToken, error) {
	var tokens []model.APIToken
	err := s.db.SelectContext(ctx, &tokens, `
		SELECT * FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	return tokens, err
}

func (s *Store) DeleteAPIToken(ctx context.Context, userID, tokenID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM api_tokens WHERE user_id = $1 AND id = $2`, userID, tokenID)
	return execOrNotFound(res, err)
}

func (s *Store) FindAPIToken(ctx context.Context, tokenValue string) (*model.APIToken, *model.User, error) {
	var row struct {
		model.APIToken
		Phone         string    `db:"phone"`
		Email         *string   `db:"email"`
		PasswordHash  string    `db:"password_hash"`
		UserName      string    `db:"user_name"`
		Timezone      string    `db:"timezone"`
		IsActive      bool      `db:"is_active"`
		UserCreatedAt time.Time `db:"user_created_at"`
		UpdatedAt     time.Time `db:"updated_at"`
	}
	err := s.db.GetContext(ctx, &row, `
		SELECT
			t.*,
			u.phone,
			u.email,
			u.password_hash,
			u.name AS user_name,
			u.timezone,
			u.is_active,
			u.created_at AS user_created_at,
			u.updated_at
		FROM api_tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token = $1
	`, tokenValue)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, apperror.ErrUnauthorized
	}
	if err != nil {
		return nil, nil, err
	}
	user := &model.User{
		ID:           row.UserID,
		Phone:        row.Phone,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Name:         row.UserName,
		Timezone:     row.Timezone,
		IsActive:     row.IsActive,
		CreatedAt:    row.UserCreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	token := row.APIToken
	return &token, user, nil
}

func (s *Store) TouchAPIToken(ctx context.Context, tokenID string) {
	_, _ = s.db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1`, tokenID)
}

func (s *Store) CreatePayment(ctx context.Context, payment model.Payment) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO payments (id, user_id, type, amount, description, proof_url, status, verified_by, verified_at, created_at)
		VALUES (:id, :user_id, :type, :amount, :description, :proof_url, :status, :verified_by, :verified_at, :created_at)
	`, payment)
	return err
}

func (s *Store) ListUserPayments(ctx context.Context, userID, status string, page, perPage int) ([]model.Payment, int64, error) {
	where := []string{"user_id = $1"}
	args := []interface{}{userID}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	return listPayments[model.Payment](ctx, s.db, "payments", where, args, page, perPage)
}

func (s *Store) ListAdminPayments(ctx context.Context, status string, page, perPage int) ([]model.PaymentWithUser, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("p.status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int64
	countQuery := `SELECT COUNT(*) FROM payments p JOIN users u ON u.id = p.user_id WHERE ` + whereSQL
	if err := s.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, perPage, offset(page, perPage))
	var items []model.PaymentWithUser
	err := s.db.SelectContext(ctx, &items, `
		SELECT p.*, u.name AS user_name, u.phone AS user_phone
		FROM payments p
		JOIN users u ON u.id = p.user_id
		WHERE `+whereSQL+`
		ORDER BY p.created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return items, total, err
}

func (s *Store) GetPayment(ctx context.Context, paymentID string) (*model.Payment, error) {
	var payment model.Payment
	err := s.db.GetContext(ctx, &payment, `SELECT * FROM payments WHERE id = $1`, paymentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &payment, err
}

func (s *Store) VerifyPayment(ctx context.Context, paymentID, adminID string, expiresAt *time.Time) (*model.Payment, *model.UserBalance, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer rollback(tx)

	var payment model.Payment
	err = tx.GetContext(ctx, &payment, `
		UPDATE payments
		SET status = 'verified', verified_by = $2, verified_at = NOW()
		WHERE id = $1 AND status = 'pending'
		RETURNING *
	`, paymentID, adminID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}

	var balance model.UserBalance
	err = tx.GetContext(ctx, &balance, `
		UPDATE user_balances
		SET balance = balance + $2,
			expires_at = COALESCE($3, expires_at),
			updated_at = NOW()
		WHERE user_id = $1
		RETURNING *
	`, payment.UserID, payment.Amount, expiresAt)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	return &payment, &balance, nil
}

func (s *Store) RejectPayment(ctx context.Context, paymentID string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE payments SET status = 'rejected' WHERE id = $1 AND status = 'pending'
	`, paymentID)
	return execOrNotFound(res, err)
}

func (s *Store) CreateTransaction(ctx context.Context, tx *model.Transaction) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO transactions (id, user_id, group_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at)
		VALUES (:id, :user_id, :group_id, :jumlah, :deskripsi, :kategori, :tipe, :source, :recorded_by, :confidence, :created_at)
	`, tx)
	return err
}

func (s *Store) ListTransactions(ctx context.Context, params model.TransactionListParams) ([]model.Transaction, int64, error) {
	where, args := transactionWhere(params)
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM transactions WHERE `+whereSQL, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, params.PerPage, offset(params.Page, params.PerPage))
	var items []model.Transaction
	err := s.db.SelectContext(ctx, &items, `
		SELECT id, user_id, group_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at
		FROM transactions
		WHERE `+whereSQL+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return items, total, err
}

func (s *Store) ListTransactionsForRange(ctx context.Context, params model.TransactionListParams, start, end time.Time) ([]model.Transaction, error) {
	params.From = &start
	params.To = &end
	params.Page = 1
	params.PerPage = 10000
	items, _, err := s.ListTransactions(ctx, params)
	return items, err
}

func (s *Store) GetTransaction(ctx context.Context, txID, userID string) (*model.Transaction, error) {
	var tx model.Transaction
	err := s.db.GetContext(ctx, &tx, `
		SELECT id, user_id, group_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at
		FROM transactions
		WHERE id = $1 AND user_id = $2
	`, txID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &tx, err
}

func (s *Store) DeleteTransaction(ctx context.Context, txID, userID string) error {
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM transactions WHERE id = $1 AND user_id = $2
	`, txID, userID)
	return execOrNotFound(res, err)
}

func (s *Store) GetTransactionSummary(ctx context.Context, userID, month string) (*model.DashboardSummary, error) {
	start, end, err := MonthRange(month)
	if err != nil {
		return nil, err
	}
	summary := &model.DashboardSummary{Month: month}
	err = s.db.QueryRowxContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN tipe = 'IN' THEN jumlah ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN tipe = 'OUT' THEN jumlah ELSE 0 END), 0),
			COUNT(*)
		FROM transactions
		WHERE user_id = $1 AND group_id IS NULL AND created_at >= $2 AND created_at < $3
	`, userID, start, end).Scan(&summary.TotalIn, &summary.TotalOut, &summary.TotalTransactions)
	if err != nil {
		return nil, err
	}
	summary.Saldo = summary.TotalIn - summary.TotalOut

	prevStart := start.AddDate(0, -1, 0)
	var prevOut int
	err = s.db.GetContext(ctx, &prevOut, `
		SELECT COALESCE(SUM(jumlah), 0)
		FROM transactions
		WHERE user_id = $1 AND tipe = 'OUT' AND group_id IS NULL AND created_at >= $2 AND created_at < $3
	`, userID, prevStart, start)
	if err != nil {
		return nil, err
	}
	summary.Comparison.PrevMonthOut = prevOut
	if prevOut > 0 {
		summary.Comparison.ChangePercent = Round1(float64(summary.TotalOut-prevOut) / float64(prevOut) * 100)
	}
	return summary, nil
}

func (s *Store) GetTransactionChartData(ctx context.Context, userID, month, timezone string) (*model.ChartData, error) {
	start, end, err := MonthRange(month)
	if err != nil {
		return nil, err
	}
	chart := &model.ChartData{Month: month}
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	err = s.db.SelectContext(ctx, &chart.ByKategori, `
		SELECT kategori, SUM(jumlah) AS total, COUNT(*) AS count
		FROM transactions
		WHERE user_id = $1 AND tipe = 'OUT' AND group_id IS NULL AND created_at >= $2 AND created_at < $3
		GROUP BY kategori
		ORDER BY total DESC
	`, userID, start, end)
	if err != nil {
		return nil, err
	}
	err = s.db.SelectContext(ctx, &chart.DailyTrend, `
		SELECT DATE(created_at AT TIME ZONE $4)::TEXT AS date,
			COALESCE(SUM(CASE WHEN tipe = 'IN' THEN jumlah ELSE 0 END), 0) AS total_in,
			COALESCE(SUM(CASE WHEN tipe = 'OUT' THEN jumlah ELSE 0 END), 0) AS total_out
		FROM transactions
		WHERE user_id = $1 AND group_id IS NULL AND created_at >= $2 AND created_at < $3
		GROUP BY DATE(created_at AT TIME ZONE $4)
		ORDER BY date
	`, userID, start, end, timezone)
	if err != nil {
		return nil, err
	}
	return chart, nil
}

func (s *Store) CreateGroupWithOwner(ctx context.Context, group model.BudgetGroup) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO budget_groups (id, name, owner_id, created_at)
		VALUES (:id, :name, :owner_id, :created_at)
	`, group)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO budget_group_members (group_id, user_id, role, joined_at)
		VALUES ($1, $2, 'owner', NOW())
	`, group.ID, group.OwnerID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CountOwnedGroups(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM budget_groups WHERE owner_id = $1`, userID)
	return count, err
}

func (s *Store) ListGroups(ctx context.Context, userID string) ([]model.GroupListItem, error) {
	var items []model.GroupListItem
	err := s.db.SelectContext(ctx, &items, `
		SELECT g.id, g.name, m.role, COUNT(all_members.user_id) AS member_count
		FROM budget_groups g
		JOIN budget_group_members m ON m.group_id = g.id AND m.user_id = $1
		JOIN budget_group_members all_members ON all_members.group_id = g.id
		GROUP BY g.id, g.name, m.role
		ORDER BY g.created_at DESC
	`, userID)
	return items, err
}

func (s *Store) GetGroup(ctx context.Context, groupID string) (*model.BudgetGroup, error) {
	var group model.BudgetGroup
	err := s.db.GetContext(ctx, &group, `SELECT * FROM budget_groups WHERE id = $1`, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &group, err
}

func (s *Store) FindGroupByNameForUser(ctx context.Context, userID, name string) (*model.BudgetGroup, error) {
	var group model.BudgetGroup
	err := s.db.GetContext(ctx, &group, `
		SELECT g.*
		FROM budget_groups g
		JOIN budget_group_members m ON m.group_id = g.id
		WHERE m.user_id = $1 AND LOWER(g.name) = LOWER($2)
		LIMIT 1
	`, userID, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &group, err
}

func (s *Store) GetMembership(ctx context.Context, groupID, userID string) (*model.BudgetGroupMember, error) {
	var member model.BudgetGroupMember
	err := s.db.GetContext(ctx, &member, `
		SELECT * FROM budget_group_members WHERE group_id = $1 AND user_id = $2
	`, groupID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrForbidden
	}
	return &member, err
}

func (s *Store) CountGroupMembers(ctx context.Context, groupID string) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM budget_group_members WHERE group_id = $1`, groupID)
	return count, err
}

func (s *Store) AddGroupMember(ctx context.Context, groupID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO budget_group_members (group_id, user_id, role, joined_at)
		VALUES ($1, $2, 'member', NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING
	`, groupID, userID)
	return err
}

func (s *Store) ListGroupMembers(ctx context.Context, groupID string) ([]model.GroupMemberView, error) {
	var members []model.GroupMemberView
	err := s.db.SelectContext(ctx, &members, `
		SELECT u.id AS user_id, u.name, u.phone, m.role
		FROM budget_group_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.group_id = $1
		ORDER BY m.joined_at ASC
	`, groupID)
	return members, err
}

func (s *Store) GetReferralCodeByUser(ctx context.Context, userID string) (*model.ReferralCode, error) {
	var code model.ReferralCode
	err := s.db.GetContext(ctx, &code, `SELECT * FROM referral_codes WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &code, err
}

func (s *Store) CreateReferralCode(ctx context.Context, code model.ReferralCode) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO referral_codes (id, user_id, code, name, phone, commission, is_active, created_at)
		VALUES (:id, :user_id, :code, :name, :phone, :commission, :is_active, :created_at)
	`, code)
	return err
}

func (s *Store) ReferralSummary(ctx context.Context, code string) (totalReferrals, activeReferrals, totalEarned, pendingPayout int, err error) {
	err = s.db.QueryRowxContext(ctx, `
		SELECT
			COUNT(rs.user_id),
			COUNT(rs.user_id) FILTER (WHERE u.is_active = TRUE),
			COALESCE(SUM(rc.commission) FILTER (WHERE rp.status = 'paid'), 0),
			COALESCE(COUNT(rs.user_id) FILTER (WHERE u.is_active = TRUE) * MAX(rc.commission), 0)
		FROM referral_codes rc
		LEFT JOIN referral_signups rs ON rs.referral_code = rc.code
		LEFT JOIN users u ON u.id = rs.user_id
		LEFT JOIN referral_payouts rp ON rp.referral_code = rc.code AND rp.status = 'paid'
		WHERE rc.code = $1
		GROUP BY rc.code
	`, code).Scan(&totalReferrals, &activeReferrals, &totalEarned, &pendingPayout)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, 0, nil
	}
	return totalReferrals, activeReferrals, totalEarned, pendingPayout, err
}

func (s *Store) CreateAdmin(ctx context.Context, admin model.Admin) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO admins (id, username, password_hash, role, created_at)
		VALUES (:id, :username, :password_hash, :role, :created_at)
		ON CONFLICT (username) DO NOTHING
	`, admin)
	return err
}

func (s *Store) GetAdminByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	err := s.db.GetContext(ctx, &admin, `SELECT * FROM admins WHERE username = $1`, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrUnauthorized
	}
	return &admin, err
}

func (s *Store) GetAdminByID(ctx context.Context, id string) (*model.Admin, error) {
	var admin model.Admin
	err := s.db.GetContext(ctx, &admin, `SELECT * FROM admins WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrUnauthorized
	}
	return &admin, err
}

func (s *Store) CreateAdminLog(ctx context.Context, log model.AdminLog) error {
	_, err := s.db.NamedExecContext(ctx, `
		INSERT INTO admin_logs (id, admin_id, action, target_type, target_id, detail, created_at)
		VALUES (:id, :admin_id, :action, :target_type, :target_id, :detail, :created_at)
	`, log)
	return err
}

func (s *Store) ListAdminLogs(ctx context.Context, adminID, action string, page, perPage int) ([]model.AdminLogView, int64, error) {
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
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM admin_logs l WHERE `+whereSQL, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, perPage, offset(page, perPage))
	var logs []model.AdminLogView
	err := s.db.SelectContext(ctx, &logs, `
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

func (s *Store) LogWAMessage(ctx context.Context, userID *string, sender, message string) {
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO wa_message_logs (id, user_id, sender, message, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, ids.New("wa"), userID, sender, message)
}

func (s *Store) AdminDashboardStats(ctx context.Context) (map[string]interface{}, error) {
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
		if err := s.db.GetContext(ctx, &value, query); err != nil {
			return nil, err
		}
		stats[key] = value
	}
	return stats, nil
}

func (s *Store) ListAdminUsers(ctx context.Context, status, search, sort, order string, page, perPage int) ([]model.UserWithBalance, int64, error) {
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
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM users u WHERE `+whereSQL, args...); err != nil {
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
	err := s.db.SelectContext(ctx, &users, `
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

func listPayments[T any](ctx context.Context, db *sqlx.DB, table string, where []string, args []interface{}, page, perPage int) ([]T, int64, error) {
	whereSQL := strings.Join(where, " AND ")
	var total int64
	if err := db.GetContext(ctx, &total, `SELECT COUNT(*) FROM `+table+` WHERE `+whereSQL, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, perPage, offset(page, perPage))
	var items []T
	err := db.SelectContext(ctx, &items, `
		SELECT * FROM `+table+`
		WHERE `+whereSQL+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return items, total, err
}

func transactionWhere(params model.TransactionListParams) ([]string, []interface{}) {
	var where []string
	var args []interface{}
	if params.GroupID == nil {
		where = []string{"user_id = $1"}
		args = []interface{}{params.UserID}
		where = append(where, "group_id IS NULL")
	} else {
		where = []string{"group_id = $1"}
		args = []interface{}{*params.GroupID}
		if params.UserID != "" {
			args = append(args, params.UserID)
			where = append(where, fmt.Sprintf("user_id = $%d", len(args)))
		}
	}
	if params.Tipe != nil && *params.Tipe != "" {
		args = append(args, *params.Tipe)
		where = append(where, fmt.Sprintf("tipe = $%d", len(args)))
	}
	if params.Kategori != nil && *params.Kategori != "" {
		args = append(args, *params.Kategori)
		where = append(where, fmt.Sprintf("kategori = $%d", len(args)))
	}
	if params.From != nil {
		args = append(args, *params.From)
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if params.To != nil {
		args = append(args, *params.To)
		where = append(where, fmt.Sprintf("created_at < $%d", len(args)))
	}
	if params.Search != nil && *params.Search != "" {
		args = append(args, "%"+*params.Search+"%")
		where = append(where, fmt.Sprintf("deskripsi ILIKE $%d", len(args)))
	}
	return where, args
}

func MonthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, apperror.New(apperror.ErrValidation, "Format bulan harus YYYY-MM")
	}
	return start, start.AddDate(0, 1, 0), nil
}

func Round1(value float64) float64 {
	return float64(int(value*10)) / 10
}

func userOrNotFound(user *model.User, err error) (*model.User, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return user, err
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

func offset(page, perPage int) int {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	return (page - 1) * perPage
}

func rollback(tx *sqlx.Tx) {
	_ = tx.Rollback()
}

func isDuplicate(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}
