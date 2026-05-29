package users

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

func (r *Repository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, id)
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

func (r *Repository) Update(ctx context.Context, userID string, name, email, timezone *string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `
		UPDATE users
		SET
			name = COALESCE($2, name),
			email = COALESCE($3, email),
			timezone = COALESCE($4, timezone),
			updated_at = NOW()
		WHERE id = $1
		RETURNING *
	`, userID, name, email, timezone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &user, err
}
