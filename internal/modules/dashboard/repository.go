package dashboard

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

func (r *Repository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	return &user, err
}

func (r *Repository) Summary(ctx context.Context, userID, month string) (*model.DashboardSummary, error) {
	start, end, err := monthRange(month)
	if err != nil {
		return nil, err
	}
	summary := &model.DashboardSummary{Month: month}
	err = r.db.QueryRowxContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN tipe = 'IN' THEN jumlah ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN tipe = 'OUT' THEN jumlah ELSE 0 END), 0),
			COUNT(*)
		FROM transactions
		WHERE user_id = $1 AND group_id IS NULL AND created_at >= $2 AND created_at < $3
	`, userID, start, end).Scan(&summary.TotalIn, &summary.TotalOut, &summary.TotalTransactions)
	if err != nil {
		return nil, err
	}
	summary.Saldo = summary.TotalIn - summary.TotalOut

	prevStart := start.AddDate(0, -1, 0)
	var prevOut int
	err = r.db.GetContext(ctx, &prevOut, `
		SELECT COALESCE(SUM(jumlah), 0)
		FROM transactions
		WHERE user_id = $1 AND tipe = 'OUT' AND group_id IS NULL AND created_at >= $2 AND created_at < $3
	`, userID, prevStart, start)
	if err != nil {
		return nil, err
	}
	summary.Comparison.PrevMonthOut = prevOut
	if prevOut > 0 {
		summary.Comparison.ChangePercent = round1(float64(summary.TotalOut-prevOut) / float64(prevOut) * 100)
	}
	return summary, nil
}

func (r *Repository) Chart(ctx context.Context, userID, month, timezone string) (*model.ChartData, error) {
	start, end, err := monthRange(month)
	if err != nil {
		return nil, err
	}
	chart := &model.ChartData{Month: month}
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	err = r.db.SelectContext(ctx, &chart.ByKategori, `
		SELECT kategori, SUM(jumlah) AS total, COUNT(*) AS count
		FROM transactions
		WHERE user_id = $1 AND tipe = 'OUT' AND group_id IS NULL AND created_at >= $2 AND created_at < $3
		GROUP BY kategori
		ORDER BY total DESC
	`, userID, start, end)
	if err != nil {
		return nil, err
	}
	err = r.db.SelectContext(ctx, &chart.DailyTrend, `
		SELECT DATE(created_at AT TIME ZONE $4)::TEXT AS date,
			COALESCE(SUM(CASE WHEN tipe = 'IN' THEN jumlah ELSE 0 END), 0) AS total_in,
			COALESCE(SUM(CASE WHEN tipe = 'OUT' THEN jumlah ELSE 0 END), 0) AS total_out
		FROM transactions
		WHERE user_id = $1 AND group_id IS NULL AND created_at >= $2 AND created_at < $3
		GROUP BY DATE(created_at AT TIME ZONE $4)
		ORDER BY date
	`, userID, start, end, timezone)
	if err != nil {
		return nil, err
	}
	return chart, nil
}

func (r *Repository) TransactionsForPeriod(ctx context.Context, userID string, start, end time.Time) ([]model.Transaction, error) {
	var items []model.Transaction
	err := r.db.SelectContext(ctx, &items, `
		SELECT id, user_id, group_id, wallet_id, jumlah, deskripsi, kategori, tipe, source, recorded_by, confidence, created_at
		FROM transactions
		WHERE user_id = $1 AND group_id IS NULL AND created_at >= $2 AND created_at < $3
		ORDER BY created_at DESC
		LIMIT 1000 OFFSET 0
	`, userID, start, end)
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
