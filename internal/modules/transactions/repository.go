package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (r *Repository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
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

func (r *Repository) GetMembership(ctx context.Context, groupID, userID string) (*model.BudgetGroupMember, error) {
	var member model.BudgetGroupMember
	err := r.db.GetContext(ctx, &member, `
		SELECT * FROM budget_group_members WHERE group_id = $1 AND user_id = $2
	`, groupID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrForbidden
	}
	return &member, err
}

func (r *Repository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *Repository) Create(ctx context.Context, tx *model.Transaction) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO transactions (id, user_id, group_id, account_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at)
		VALUES (:id, :user_id, :group_id, :account_id, :jumlah, :deskripsi, :kategori, :tipe, :source, :recorded_by, :confidence, :created_at)
	`, tx)
	return err
}

func (r *Repository) CreateTx(ctx context.Context, dbTx *sqlx.Tx, tx *model.Transaction) error {
	_, err := dbTx.NamedExecContext(ctx, `
		INSERT INTO transactions (id, user_id, group_id, account_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at)
		VALUES (:id, :user_id, :group_id, :account_id, :jumlah, :deskripsi, :kategori, :tipe, :source, :recorded_by, :confidence, :created_at)
	`, tx)
	return err
}

func (r *Repository) List(ctx context.Context, params ListParams) ([]model.Transaction, int64, error) {
	where, args := transactionWhere(params)
	whereSQL := strings.Join(where, " AND ")

	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM transactions WHERE `+whereSQL, args...); err != nil {
		return nil, 0, err
	}

	args = append(args, params.PerPage, offset(params.Page, params.PerPage))
	var items []model.Transaction
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, user_id, group_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at
		FROM transactions
		WHERE `+whereSQL+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
		args...,
	)
	return items, total, err
}

func (r *Repository) Get(ctx context.Context, txID, userID string) (*model.Transaction, error) {
	var tx model.Transaction
	err := r.db.GetContext(ctx, &tx, `
		SELECT id, user_id, group_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at
		FROM transactions
		WHERE id = $1 AND user_id = $2
	`, txID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &tx, err
}

// Delete fetches the transaction first (to return its data for balance reversal), then deletes it.
func (r *Repository) Delete(ctx context.Context, txID, userID string) (*model.Transaction, error) {
	tx, err := r.Get(ctx, txID, userID)
	if err != nil {
		return nil, err
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2`, txID, userID)
	return tx, err
}

func (r *Repository) DeleteTx(ctx context.Context, dbTx *sqlx.Tx, txID, userID string) error {
	res, err := dbTx.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2`, txID, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func transactionWhere(params ListParams) ([]string, []interface{}) {
	var where []string
	var args []interface{}
	if params.GroupID == nil {
		where = []string{"user_id = $1"}
		args = []interface{}{params.UserID}
		where = append(where, "group_id IS NULL")
	} else {
		where = []string{"group_id = $1"}
		args = []interface{}{*params.GroupID}
		if params.UserID != "" {
			args = append(args, params.UserID)
			where = append(where, fmt.Sprintf("user_id = $%d", len(args)))
		}
	}
	if params.Tipe != nil && *params.Tipe != "" {
		args = append(args, *params.Tipe)
		where = append(where, fmt.Sprintf("tipe = $%d", len(args)))
	}
	if params.Kategori != nil && *params.Kategori != "" {
		args = append(args, *params.Kategori)
		where = append(where, fmt.Sprintf("kategori = $%d", len(args)))
	}
	if params.From != nil {
		args = append(args, *params.From)
		where = append(where, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if params.To != nil {
		args = append(args, *params.To)
		where = append(where, fmt.Sprintf("created_at < $%d", len(args)))
	}
	if params.Search != nil && *params.Search != "" {
		args = append(args, "%"+*params.Search+"%")
		where = append(where, fmt.Sprintf("deskripsi ILIKE $%d", len(args)))
	}
	return where, args
}

func offset(page, perPage int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * perPage
}
