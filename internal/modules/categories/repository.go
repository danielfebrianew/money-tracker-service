package categories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (r *Repository) List(ctx context.Context, userID string) ([]model.Category, error) {
	var items []model.Category
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, user_id, name, description, icon, color, is_default, created_at
		FROM categories
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at ASC
	`, userID)
	return items, err
}

func (r *Repository) Get(ctx context.Context, id, userID string) (*model.Category, error) {
	var item model.Category
	err := r.db.GetContext(ctx, &item, `
		SELECT id, user_id, name, description, icon, color, is_default, created_at
		FROM categories
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &item, err
}

func (r *Repository) GetByName(ctx context.Context, userID, name string) (*model.Category, error) {
	var item model.Category
	err := r.db.GetContext(ctx, &item, `
		SELECT id, user_id, name, description, icon, color, is_default, created_at
		FROM categories
		WHERE user_id = $1 AND name = $2
	`, userID, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &item, err
}

func (r *Repository) Create(ctx context.Context, cat *model.Category) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO categories (id, user_id, name, description, icon, color, is_default, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, cat.ID, cat.UserID, cat.Name, cat.Description, cat.Icon, cat.Color, cat.IsDefault, cat.CreatedAt)
	return err
}

func (r *Repository) BulkCreate(ctx context.Context, cats []model.Category) error {
	if len(cats) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(cats))
	args := make([]interface{}, 0, len(cats)*8)
	for i, cat := range cats {
		base := i * 8
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8,
		))
		args = append(args, cat.ID, cat.UserID, cat.Name, cat.Description, cat.Icon, cat.Color, cat.IsDefault, cat.CreatedAt)
	}
	query := `INSERT INTO categories (id, user_id, name, description, icon, color, is_default, created_at)
		VALUES ` + strings.Join(placeholders, ",") + ` ON CONFLICT (user_id, name) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) Update(ctx context.Context, cat *model.Category) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE categories SET description = $1, icon = $2, color = $3 WHERE id = $4 AND user_id = $5
	`, cat.Description, cat.Icon, cat.Color, cat.ID, cat.UserID)
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
	res, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *Repository) UsedInTransactions(ctx context.Context, userID, name string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM transactions WHERE user_id = $1 AND kategori = $2`, userID, name)
	return count > 0, err
}

func (r *Repository) UsedInBudgets(ctx context.Context, userID, name string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM budgets WHERE user_id = $1 AND kategori = $2`, userID, name)
	return count > 0, err
}
