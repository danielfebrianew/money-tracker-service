package tokens

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

func (r *Repository) Create(ctx context.Context, token model.APIToken) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO api_tokens (id, user_id, token, name, last_used_at, created_at)
		VALUES (:id, :user_id, :token, :name, :last_used_at, :created_at)
	`, token)
	return err
}

func (r *Repository) Count(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM api_tokens WHERE user_id = $1`, userID)
	return count, err
}

func (r *Repository) List(ctx context.Context, userID string) ([]model.APIToken, error) {
	var tokens []model.APIToken
	err := r.db.SelectContext(ctx, &tokens, `
		SELECT * FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	return tokens, err
}

func (r *Repository) Delete(ctx context.Context, userID, tokenID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM api_tokens WHERE user_id = $1 AND id = $2`, userID, tokenID)
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

func (r *Repository) Find(ctx context.Context, tokenValue string) (*model.APIToken, *model.User, error) {
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
	err := r.db.GetContext(ctx, &row, `
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

func (r *Repository) Touch(ctx context.Context, tokenID string) {
	_, _ = r.db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1`, tokenID)
}
