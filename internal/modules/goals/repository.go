package goals

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

func (r *Repository) List(ctx context.Context, userID string) ([]model.Goal, error) {
	var items []model.Goal
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, user_id, name, target_amount, current_amount, deadline, icon, color, status, created_at, updated_at
		FROM goals
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	return items, err
}

func (r *Repository) Get(ctx context.Context, id, userID string) (*model.Goal, error) {
	var goal model.Goal
	err := r.db.GetContext(ctx, &goal, `
		SELECT id, user_id, name, target_amount, current_amount, deadline, icon, color, status, created_at, updated_at
		FROM goals
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &goal, err
}

func (r *Repository) Create(ctx context.Context, goal *model.Goal) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO goals (id, user_id, name, target_amount, current_amount, deadline, icon, color, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, goal.ID, goal.UserID, goal.Name, goal.TargetAmount, goal.CurrentAmount,
		goal.Deadline, goal.Icon, goal.Color, goal.Status, goal.CreatedAt, goal.UpdatedAt)
	return err
}

func (r *Repository) Update(ctx context.Context, goal *model.Goal) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE goals
		SET name = $1, target_amount = $2, current_amount = $3, deadline = $4, icon = $5, color = $6, status = $7, updated_at = $8
		WHERE id = $9 AND user_id = $10
	`, goal.Name, goal.TargetAmount, goal.CurrentAmount, goal.Deadline,
		goal.Icon, goal.Color, goal.Status, goal.UpdatedAt, goal.ID, goal.UserID)
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
	res, err := r.db.ExecContext(ctx, `DELETE FROM goals WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
