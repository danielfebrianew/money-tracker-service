package budget

import (
	"context"
	"database/sql"
	"errors"

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

func (r *Repository) List(ctx context.Context, userID, month string) ([]model.BudgetWithSpent, error) {
	var items []model.BudgetWithSpent
	err := r.db.SelectContext(ctx, &items, `
		SELECT b.id, b.user_id, b.kategori, b.limit, b.month, b.created_at, b.updated_at,
		       COALESCE(SUM(t.jumlah), 0) AS spent
		FROM budgets b
		LEFT JOIN transactions t
		       ON t.user_id = b.user_id
		      AND t.kategori = b.kategori
		      AND t.tipe = 'OUT'
		      AND to_char(t.created_at AT TIME ZONE 'UTC', 'YYYY-MM') = b.month
		WHERE b.user_id = $1 AND b.month = $2
		GROUP BY b.id
		ORDER BY b.created_at ASC
	`, userID, month)
	return items, err
}

func (r *Repository) Get(ctx context.Context, id, userID string) (*model.BudgetWithSpent, error) {
	var item model.BudgetWithSpent
	err := r.db.GetContext(ctx, &item, `
		SELECT b.id, b.user_id, b.kategori, b.limit, b.month, b.created_at, b.updated_at,
		       COALESCE(SUM(t.jumlah), 0) AS spent
		FROM budgets b
		LEFT JOIN transactions t
		       ON t.user_id = b.user_id
		      AND t.kategori = b.kategori
		      AND t.tipe = 'OUT'
		      AND to_char(t.created_at AT TIME ZONE 'UTC', 'YYYY-MM') = b.month
		WHERE b.id = $1 AND b.user_id = $2
		GROUP BY b.id
	`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &item, err
}

func (r *Repository) ExistsForMonth(ctx context.Context, userID, kategori, month string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM budgets WHERE user_id = $1 AND kategori = $2 AND month = $3
	`, userID, kategori, month)
	return count > 0, err
}

func (r *Repository) Create(ctx context.Context, b *model.Budget) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO budgets (id, user_id, kategori, "limit", month, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, b.ID, b.UserID, b.Kategori, b.Limit, b.Month, b.CreatedAt, b.UpdatedAt)
	return err
}

func (r *Repository) Update(ctx context.Context, id, userID string, limit int) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE budgets SET "limit" = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3
	`, limit, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM budgets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *Repository) History(ctx context.Context, userID, kategori string, months int) ([]model.BudgetHistory, error) {
	var items []model.BudgetHistory
	var err error
	if kategori != "" {
		err = r.db.SelectContext(ctx, &items, `
			SELECT to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM') AS month,
			       kategori,
			       COALESCE(SUM(jumlah), 0) AS total_spent
			FROM transactions
			WHERE user_id = $1
			  AND kategori = $2
			  AND tipe = 'OUT'
			  AND created_at >= date_trunc('month', NOW()) - ($3 - 1) * INTERVAL '1 month'
			GROUP BY month, kategori
			ORDER BY month ASC
		`, userID, kategori, months)
	} else {
		err = r.db.SelectContext(ctx, &items, `
			SELECT to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM') AS month,
			       kategori,
			       COALESCE(SUM(jumlah), 0) AS total_spent
			FROM transactions
			WHERE user_id = $1
			  AND tipe = 'OUT'
			  AND created_at >= date_trunc('month', NOW()) - ($2 - 1) * INTERVAL '1 month'
			GROUP BY month, kategori
			ORDER BY month ASC, kategori ASC
		`, userID, months)
	}
	return items, err
}
