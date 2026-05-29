package dashboard

import (
	"context"
	"time"

	"money-tracker-service/internal/cache"
	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
)

type Service struct {
	cache      *cache.Cache
	repository *Repository
}

func NewService(cache *cache.Cache, repository *Repository) *Service {
	return &Service{cache: cache, repository: repository}
}

func (s *Service) Summary(ctx context.Context, userID, month string) (*model.DashboardSummary, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	key := "dash:" + userID + ":summary:" + month
	var cached model.DashboardSummary
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}
	summary, err := s.repository.Summary(ctx, userID, month)
	if err != nil {
		return nil, err
	}
	s.cache.SetJSON(ctx, key, summary, 5*time.Minute)
	return summary, nil
}

func (s *Service) Chart(ctx context.Context, userID, month string) (*model.ChartData, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	key := "dash:" + userID + ":chart:" + month
	var cached model.ChartData
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	chart, err := s.repository.Chart(ctx, userID, month, user.Timezone)
	if err != nil {
		return nil, err
	}
	s.cache.SetJSON(ctx, key, chart, 5*time.Minute)
	return chart, nil
}

func (s *Service) Report(ctx context.Context, userID, period, date string) (*model.PeriodReport, error) {
	if period == "" {
		period = "monthly"
	}
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	key := "report:" + userID + ":" + period + ":" + date
	var cached model.PeriodReport
	if s.cache.GetJSON(ctx, key, &cached) {
		return &cached, nil
	}
	start, end, err := periodRange(period, date)
	if err != nil {
		return nil, err
	}
	items, err := s.repository.TransactionsForPeriod(ctx, userID, start, end)
	if err != nil {
		return nil, err
	}
	report := &model.PeriodReport{
		Period:       period,
		StartDate:    start.Format("2006-01-02"),
		EndDate:      end.AddDate(0, 0, -1).Format("2006-01-02"),
		Transactions: items,
	}
	categories := map[string]model.CategoryTotal{}
	for _, item := range items {
		if item.Tipe == "IN" {
			report.TotalIn += item.Jumlah
			continue
		}
		report.TotalOut += item.Jumlah
		cat := categories[item.Kategori]
		cat.Kategori = item.Kategori
		cat.Total += item.Jumlah
		cat.Count++
		categories[item.Kategori] = cat
	}
	report.Saldo = report.TotalIn - report.TotalOut
	for _, cat := range categories {
		report.ByKategori = append(report.ByKategori, cat)
	}
	s.cache.SetJSON(ctx, key, report, 10*time.Minute)
	return report, nil
}

func periodRange(period, date string) (time.Time, time.Time, error) {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		if month, monthErr := time.Parse("2006-01", date); monthErr == nil {
			parsed = month
		} else {
			return time.Time{}, time.Time{}, apperror.New(apperror.ErrValidation, "Format tanggal harus YYYY-MM-DD")
		}
	}
	switch period {
	case "daily":
		start := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 0, 1), nil
	case "weekly":
		weekday := int(parsed.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
		return start, start.AddDate(0, 0, 7), nil
	case "monthly":
		start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), nil
	default:
		return time.Time{}, time.Time{}, apperror.New(apperror.ErrValidation, "Period harus daily, weekly, atau monthly")
	}
}
