package payments

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
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, payment model.Payment) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO payments (id, user_id, type, amount, description, proof_url, status, verified_by, verified_at, created_at)
		VALUES (:id, :user_id, :type, :amount, :description, :proof_url, :status, :verified_by, :verified_at, :created_at)
	`, payment)
	return err
}

func (r *Repository) ListUser(ctx context.Context, userID, status string, page, perPage int) ([]model.Payment, int64, error) {
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
		SELECT *
		FROM payments
		WHERE `+whereSQL+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return items, total, err
}

func (r *Repository) ListAdmin(ctx context.Context, status string, page, perPage int) ([]model.PaymentWithUser, int64, error) {
	where := []string{"1=1"}
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("p.status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	countQuery := `SELECT COUNT(*) FROM payments p JOIN users u ON u.id = p.user_id WHERE ` + whereSQL
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, perPage, offset(page, perPage))
	var items []model.PaymentWithUser
	err := r.db.SelectContext(ctx, &items, `
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

func (r *Repository) Get(ctx context.Context, paymentID string) (*model.Payment, error) {
	var payment model.Payment
	err := r.db.GetContext(ctx, &payment, `SELECT * FROM payments WHERE id = $1`, paymentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &payment, err
}

func (r *Repository) Verify(ctx context.Context, paymentID, adminID string, expiresAt *time.Time) (*model.Payment, *model.UserBalance, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
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

func (r *Repository) Reject(ctx context.Context, paymentID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE payments SET status = 'rejected' WHERE id = $1 AND status = 'pending'
	`, paymentID)
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
	return (page - 1) * perPage
}

func rollback(tx *sqlx.Tx) {
	_ = tx.Rollback()
}
