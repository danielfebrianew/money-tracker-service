package referral

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

func (r *Repository) GetCodeByUser(ctx context.Context, userID string) (*model.ReferralCode, error) {
	var code model.ReferralCode
	err := r.db.GetContext(ctx, &code, `SELECT * FROM referral_codes WHERE user_id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &code, err
}

func (r *Repository) CreateCode(ctx context.Context, code model.ReferralCode) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO referral_codes (id, user_id, code, name, phone, commission, is_active, created_at)
		VALUES (:id, :user_id, :code, :name, :phone, :commission, :is_active, :created_at)
	`, code)
	return err
}

func (r *Repository) Summary(ctx context.Context, code string) (totalReferrals, activeReferrals, totalEarned, pendingPayout int, err error) {
	err = r.db.QueryRowxContext(ctx, `
		SELECT
			COUNT(rs.user_id),
			COUNT(rs.user_id) FILTER (WHERE u.is_active = TRUE),
			COALESCE(SUM(rc.commission) FILTER (WHERE rp.status = 'paid'), 0),
			COALESCE(COUNT(rs.user_id) FILTER (WHERE u.is_active = TRUE) * MAX(rc.commission), 0)
		FROM referral_codes rc
		LEFT JOIN referral_signups rs ON rs.referral_code = rc.code
		LEFT JOIN users u ON u.id = rs.user_id
		LEFT JOIN referral_payouts rp ON rp.referral_code = rc.code AND rp.status = 'paid'
		WHERE rc.code = $1
		GROUP BY rc.code
	`, code).Scan(&totalReferrals, &activeReferrals, &totalEarned, &pendingPayout)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, 0, nil
	}
	return totalReferrals, activeReferrals, totalEarned, pendingPayout, err
}

func (r *Repository) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &user, err
}
