package balance

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context, userID string) (*model.UserBalance, error) {
	var balance model.UserBalance
	err := r.db.GetContext(ctx, &balance, `SELECT * FROM user_balances WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &balance, err
}

func (r *Repository) Add(ctx context.Context, userID string, amount int, expiresAt *time.Time) (*model.UserBalance, error) {
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

func (r *Repository) DeductMonthly(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE user_balances
		SET balance = GREATEST(balance - 30000, 0), updated_at = NOW()
		WHERE plan_type = 'monthly' AND balance >= 30000
	`)
	return err
}

func (r *Repository) SuspendExpiredUsers(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users
		SET is_active = FALSE, updated_at = NOW()
		WHERE id IN (
			SELECT u.id
			FROM users u
			JOIN user_balances b ON b.user_id = u.id
			WHERE u.is_active = TRUE
			  AND b.expires_at IS NOT NULL
			  AND b.expires_at < NOW() - INTERVAL '3 days'
		)
	`)
	return err
}
