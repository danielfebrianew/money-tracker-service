package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"

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

func (r *Repository) CreateUserWithBalance(ctx context.Context, user *model.User, referralCode *string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
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
			if strings.Contains(err.Error(), "email") {
				return apperror.New(apperror.ErrConflict, "Email sudah terdaftar")
			}
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

func (r *Repository) GetUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE phone = $1`, phone)
	return userOrNotFound(&user, err)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE email = $1`, email)
	return userOrNotFound(&user, err)
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, id)
	return userOrNotFound(&user, err)
}

func (r *Repository) GetBalance(ctx context.Context, userID string) (*model.UserBalance, error) {
	var balance model.UserBalance
	err := r.db.GetContext(ctx, &balance, `SELECT * FROM user_balances WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &balance, err
}

func (r *Repository) CreateRefreshToken(ctx context.Context, token model.RefreshToken) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES (:id, :user_id, :token_hash, :expires_at, :created_at)
	`, token)
	return err
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.db.GetContext(ctx, &token, `SELECT * FROM refresh_tokens WHERE token_hash = $1`, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrUnauthorized
	}
	return &token, err
}

func (r *Repository) DeleteRefreshTokenByHash(ctx context.Context, hash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, hash)
	return err
}

func (r *Repository) PruneRefreshTokens(ctx context.Context, userID string, keep int) error {
	if keep < 0 {
		keep = 0
	}
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM refresh_tokens
		WHERE user_id = $1
		  AND id NOT IN (
			SELECT id
			FROM refresh_tokens
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		  )
	`, userID, keep)
	return err
}

func (r *Repository) CreateAdminRefreshToken(ctx context.Context, token model.AdminRefreshToken) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO admin_refresh_tokens (id, admin_id, token_hash, expires_at, created_at)
		VALUES (:id, :admin_id, :token_hash, :expires_at, :created_at)
	`, token)
	return err
}

func (r *Repository) GetAdminRefreshTokenByHash(ctx context.Context, hash string) (*model.AdminRefreshToken, error) {
	var token model.AdminRefreshToken
	err := r.db.GetContext(ctx, &token, `SELECT * FROM admin_refresh_tokens WHERE token_hash = $1`, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrUnauthorized
	}
	return &token, err
}

func (r *Repository) DeleteAdminRefreshTokenByHash(ctx context.Context, hash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admin_refresh_tokens WHERE token_hash = $1`, hash)
	return err
}

func (r *Repository) PruneAdminRefreshTokens(ctx context.Context, adminID string, keep int) error {
	if keep < 0 {
		keep = 0
	}
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM admin_refresh_tokens
		WHERE admin_id = $1
		  AND id NOT IN (
			SELECT id
			FROM admin_refresh_tokens
			WHERE admin_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		  )
	`, adminID, keep)
	return err
}

func (r *Repository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, userID, passwordHash)
	return execOrNotFound(res, err)
}

func (r *Repository) GetAdminByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.GetContext(ctx, &admin, `SELECT * FROM admins WHERE username = $1`, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrUnauthorized
	}
	return &admin, err
}

func (r *Repository) CreateAdmin(ctx context.Context, admin model.Admin) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO admins (id, username, password_hash, role, created_at)
		VALUES (:id, :username, :password_hash, :role, :created_at)
		ON CONFLICT (username) DO NOTHING
	`, admin)
	return err
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

func rollback(tx *sqlx.Tx) {
	_ = tx.Rollback()
}

func isDuplicate(err error) bool {
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint")
}
