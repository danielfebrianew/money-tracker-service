package categories

import (
	"context"
	"strings"
	"time"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/ids"
)

var defaultCategories = []struct {
	Name, Description, Icon, Color string
}{
	{"Makan", "Makanan & Minuman", "🍽️", "#FF5722"},
	{"Transport", "Transportasi", "🚗", "#2196F3"},
	{"Tagihan", "Tagihan & Utilitas", "💡", "#FF9800"},
	{"Belanja", "Belanja", "🛍️", "#9C27B0"},
	{"Pemasukan", "Pemasukan", "💰", "#4CAF50"},
	{"Lainnya", "Lainnya", "🏷️", "#607D8B"},
}

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, userID string) ([]model.Category, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (*model.Category, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 50 {
		return nil, apperror.New(apperror.ErrValidation, "Nama kategori wajib diisi dan maksimal 50 karakter")
	}
	description := strings.TrimSpace(input.Description)
	if description == "" {
		description = name
	}
	existing, _ := s.repository.GetByName(ctx, userID, name)
	if existing != nil {
		return nil, apperror.New(apperror.ErrConflict, "Kategori dengan nama ini sudah ada")
	}
	cat := &model.Category{
		ID:          ids.New("cat"),
		UserID:      userID,
		Name:        name,
		Description: description,
		Icon:        strings.TrimSpace(input.Icon),
		Color:     strings.TrimSpace(input.Color),
		IsDefault: false,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *Service) Update(ctx context.Context, id, userID string, input UpdateInput) (*model.Category, error) {
	cat, err := s.repository.Get(ctx, id, userID)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "Kategori tidak ditemukan")
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		if description != "" {
			cat.Description = description
		}
	}
	if input.Icon != nil {
		cat.Icon = strings.TrimSpace(*input.Icon)
	}
	if input.Color != nil {
		cat.Color = strings.TrimSpace(*input.Color)
	}
	if err := s.repository.Update(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *Service) Delete(ctx context.Context, id, userID string) error {
	cat, err := s.repository.Get(ctx, id, userID)
	if err != nil {
		return apperror.New(apperror.ErrNotFound, "Kategori tidak ditemukan")
	}
	usedTx, err := s.repository.UsedInTransactions(ctx, userID, cat.Name)
	if err != nil {
		return err
	}
	if usedTx {
		return apperror.New(apperror.ErrConflict, "Kategori masih digunakan oleh transaksi")
	}
	usedBudget, err := s.repository.UsedInBudgets(ctx, userID, cat.Name)
	if err != nil {
		return err
	}
	if usedBudget {
		return apperror.New(apperror.ErrConflict, "Kategori masih digunakan oleh budget")
	}
	return s.repository.Delete(ctx, id, userID)
}

func (s *Service) SeedDefaults(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	cats := make([]model.Category, 0, len(defaultCategories))
	for _, d := range defaultCategories {
		cats = append(cats, model.Category{
			ID:        ids.New("cat"),
			UserID:    userID,
			Name:      d.Name,
			Description: d.Description,
			Icon:      d.Icon,
			Color:     d.Color,
			IsDefault: true,
			CreatedAt: now,
		})
	}
	return s.repository.BulkCreate(ctx, cats)
}

func (s *Service) GetByName(ctx context.Context, userID, name string) (*model.Category, error) {
	return s.repository.GetByName(ctx, userID, name)
}

func (s *Service) IsValidForUser(ctx context.Context, userID, name string) (bool, error) {
	_, err := s.repository.GetByName(ctx, userID, name)
	if err != nil {
		return false, nil
	}
	return true, nil
}
