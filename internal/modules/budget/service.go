package budget

import (
	"context"
	"strings"
	"time"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/ids"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, userID, month string) ([]model.BudgetWithSpent, error) {
	month = normalizeMonth(month)
	return s.repository.List(ctx, userID, month)
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (*model.BudgetWithSpent, error) {
	input.Kategori = strings.TrimSpace(input.Kategori)
	if input.Kategori == "" {
		return nil, apperror.New(apperror.ErrValidation, "Kategori wajib diisi")
	}
	if input.Limit <= 0 {
		return nil, apperror.New(apperror.ErrValidation, "Limit harus lebih dari 0")
	}
	month := normalizeMonth(input.Month)
	exists, err := s.repository.ExistsForMonth(ctx, userID, input.Kategori, month)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.New(apperror.ErrConflict, "Budget untuk kategori ini di bulan tersebut sudah ada")
	}
	now := time.Now().UTC()
	b := &model.Budget{
		ID:        ids.New("bgt"),
		UserID:    userID,
		Kategori:  input.Kategori,
		Limit:     input.Limit,
		Month:     month,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repository.Create(ctx, b); err != nil {
		return nil, err
	}
	return s.repository.Get(ctx, b.ID, userID)
}

func (s *Service) Update(ctx context.Context, userID, id string, limit int) (*model.BudgetWithSpent, error) {
	if limit <= 0 {
		return nil, apperror.New(apperror.ErrValidation, "Limit harus lebih dari 0")
	}
	if err := s.repository.Update(ctx, id, userID, limit); err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "Budget tidak ditemukan")
	}
	return s.repository.Get(ctx, id, userID)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if err := s.repository.Delete(ctx, id, userID); err != nil {
		return apperror.New(apperror.ErrNotFound, "Budget tidak ditemukan")
	}
	return nil
}

func (s *Service) History(ctx context.Context, userID, kategori string, months int) ([]model.BudgetHistory, error) {
	if months <= 0 || months > 12 {
		months = 3
	}
	return s.repository.History(ctx, userID, kategori, months)
}

func normalizeMonth(month string) string {
	month = strings.TrimSpace(month)
	if month == "" {
		return time.Now().UTC().Format("2006-01")
	}
	if _, err := time.Parse("2006-01", month); err != nil {
		return time.Now().UTC().Format("2006-01")
	}
	return month
}

