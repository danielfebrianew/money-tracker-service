package webhook

import (
	"context"
	"database/sql"
	"errors"
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

func (r *Repository) GetUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE phone = $1`, phone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &user, err
}

func (r *Repository) LogMessage(ctx context.Context, userID *string, sender, message string) {
	_, _ = r.db.ExecContext(ctx, `
		INSERT INTO wa_message_logs (id, user_id, sender, message, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, ids.New("wa"), userID, sender, message)
}

func (r *Repository) GetBalance(ctx context.Context, userID string) (*model.UserBalance, error) {
	var balance model.UserBalance
	err := r.db.GetContext(ctx, &balance, `SELECT * FROM user_balances WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &balance, err
}

func (r *Repository) TransactionSummary(ctx context.Context, userID, month string) (*model.DashboardSummary, error) {
	start, end, err := monthRange(month)
	if err != nil {
		return nil, err
	}
	summary := &model.DashboardSummary{Month: month}
	err = r.db.QueryRowxContext(ctx, `
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
	return summary, nil
}

func (r *Repository) FindGroupByNameForUser(ctx context.Context, userID, name string) (*model.BudgetGroup, error) {
	var group model.BudgetGroup
	err := r.db.GetContext(ctx, &group, `
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

func monthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, apperror.New(apperror.ErrValidation, "Format bulan harus YYYY-MM")
	}
	return start, start.AddDate(0, 1, 0), nil
}
