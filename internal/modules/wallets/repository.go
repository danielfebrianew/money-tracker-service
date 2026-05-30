package wallets

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

func (r *Repository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *Repository) List(ctx context.Context, userID string) ([]model.Wallet, error) {
	var items []model.Wallet
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, user_id, name, type, balance, icon, color, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) Get(ctx context.Context, id, userID string) (*model.Wallet, error) {
	var account model.Wallet
	err := r.db.GetContext(ctx, &account, `
		SELECT id, user_id, name, type, balance, icon, color, created_at, updated_at
		FROM wallets
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &account, err
}

func (r *Repository) Create(ctx context.Context, account *model.Wallet) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO wallets (id, user_id, name, type, balance, icon, color, created_at, updated_at)
		VALUES (:id, :user_id, :name, :type, :balance, :icon, :color, :created_at, :updated_at)
	`, account)
	return err
}

func (r *Repository) Update(ctx context.Context, account *model.Wallet) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE wallets SET name = $1, icon = $2, color = $3, updated_at = $4 WHERE id = $5 AND user_id = $6
	`, account.Name, account.Icon, account.Color, account.UpdatedAt, account.ID, account.UserID)
	return err
}

func (r *Repository) CountTransactions(ctx context.Context, walletID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM transactions WHERE wallet_id = $1`, walletID)
	return count, err
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM wallets WHERE id = $1 AND user_id = $2`, id, userID)
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

func (r *Repository) UpdateBalance(ctx context.Context, tx *sqlx.Tx, walletID string, delta int) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE wallets SET balance = balance + $1 WHERE id = $2
	`, delta, walletID)
	return err
}
