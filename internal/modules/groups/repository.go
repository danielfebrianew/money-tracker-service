package groups

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

func (r *Repository) CreateWithOwner(ctx context.Context, group model.BudgetGroup) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO budget_groups (id, name, owner_id, created_at)
		VALUES (:id, :name, :owner_id, :created_at)
	`, group)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO budget_group_members (group_id, user_id, role, joined_at)
		VALUES ($1, $2, 'owner', NOW())
	`, group.ID, group.OwnerID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) CountOwned(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM budget_groups WHERE owner_id = $1`, userID)
	return count, err
}

func (r *Repository) List(ctx context.Context, userID string) ([]model.GroupListItem, error) {
	var items []model.GroupListItem
	err := r.db.SelectContext(ctx, &items, `
		SELECT g.id, g.name, m.role, COUNT(all_members.user_id) AS member_count
		FROM budget_groups g
		JOIN budget_group_members m ON m.group_id = g.id AND m.user_id = $1
		JOIN budget_group_members all_members ON all_members.group_id = g.id
		GROUP BY g.id, g.name, m.role
		ORDER BY g.created_at DESC
	`, userID)
	return items, err
}

func (r *Repository) Get(ctx context.Context, groupID string) (*model.BudgetGroup, error) {
	var group model.BudgetGroup
	err := r.db.GetContext(ctx, &group, `SELECT * FROM budget_groups WHERE id = $1`, groupID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &group, err
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

func (r *Repository) CountMembers(ctx context.Context, groupID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM budget_group_members WHERE group_id = $1`, groupID)
	return count, err
}

func (r *Repository) AddMember(ctx context.Context, groupID, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO budget_group_members (group_id, user_id, role, joined_at)
		VALUES ($1, $2, 'member', NOW())
		ON CONFLICT (group_id, user_id) DO NOTHING
	`, groupID, userID)
	return err
}

func (r *Repository) ListMembers(ctx context.Context, groupID string) ([]model.GroupMemberView, error) {
	var members []model.GroupMemberView
	err := r.db.SelectContext(ctx, &members, `
		SELECT u.id AS user_id, u.name, u.phone, m.role
		FROM budget_group_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.group_id = $1
		ORDER BY m.joined_at ASC
	`, groupID)
	return members, err
}

func (r *Repository) GetUserByPhone(ctx context.Context, phone string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE phone = $1`, phone)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &user, err
}

func (r *Repository) ListTransactionsForReport(ctx context.Context, groupID string, start, end time.Time) ([]model.Transaction, error) {
	var items []model.Transaction
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, user_id, group_id, account_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at
		FROM transactions
		WHERE group_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at DESC
		LIMIT 10000 OFFSET 0
	`, groupID, start, end)
	return items, err
}

func monthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, apperror.New(apperror.ErrValidation, "Format bulan harus YYYY-MM")
	}
	return start, start.AddDate(0, 1, 0), nil
}

func round1(value float64) float64 {
	return float64(int(value*10)) / 10
}

func rollback(tx *sqlx.Tx) {
	_ = tx.Rollback()
}
