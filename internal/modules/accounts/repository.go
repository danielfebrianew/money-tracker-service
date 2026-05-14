package accounts

import (
	"context"
	"database/sql"
	"errors"

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

func (r *Repository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *Repository) List(ctx context.Context, userID string) ([]model.Account, error) {
	var items []model.Account
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, user_id, name, type, balance, created_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) Get(ctx context.Context, id, userID string) (*model.Account, error) {
	var account model.Account
	err := r.db.GetContext(ctx, &account, `
		SELECT id, user_id, name, type, balance, created_at
		FROM accounts
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &account, err
}

func (r *Repository) Create(ctx context.Context, account *model.Account) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO accounts (id, user_id, name, type, balance, created_at)
		VALUES (:id, :user_id, :name, :type, :balance, :created_at)
	`, account)
	return err
}

func (r *Repository) Update(ctx context.Context, account *model.Account) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE accounts SET name = $1, type = $2 WHERE id = $3 AND user_id = $4
	`, account.Name, account.Type, account.ID, account.UserID)
	return err
}

func (r *Repository) CountTransactions(ctx context.Context, accountID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM transactions WHERE account_id = $1`, accountID)
	return count, err
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = $1 AND user_id = $2`, id, userID)
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

func (r *Repository) UpdateBalance(ctx context.Context, tx *sqlx.Tx, accountID string, delta int) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE accounts SET balance = balance + $1 WHERE id = $2
	`, delta, accountID)
	return err
}
