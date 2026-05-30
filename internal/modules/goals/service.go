package goals

import (
	"context"
	"strings"
	"time"

	"money-tracker-service/internal/model"
	"money-tracker-service/internal/pkg/apperror"
	"money-tracker-service/internal/pkg/ids"
)

const (
	statusActive   = "active"
	statusAchieved = "achieved"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context, userID string) ([]model.Goal, error) {
	return s.repository.List(ctx, userID)
}

func (s *Service) Get(ctx context.Context, id, userID string) (*model.Goal, error) {
	goal, err := s.repository.Get(ctx, id, userID)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "Target tidak ditemukan")
	}
	return goal, nil
}

func (s *Service) Create(ctx context.Context, userID string, input CreateInput) (*model.Goal, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 100 {
		return nil, apperror.New(apperror.ErrValidation, "Nama target wajib diisi dan maksimal 100 karakter")
	}
	if input.TargetAmount <= 0 {
		return nil, apperror.New(apperror.ErrValidation, "Nominal target harus lebih dari 0")
	}
	deadline := strings.TrimSpace(input.Deadline)
	if deadline == "" {
		return nil, apperror.New(apperror.ErrValidation, "Deadline wajib diisi")
	}
	dl, err := time.Parse("2006-01-02", deadline)
	if err != nil {
		return nil, apperror.New(apperror.ErrValidation, "Format deadline harus YYYY-MM-DD")
	}
	if dl.Before(time.Now().Truncate(24 * time.Hour)) {
		return nil, apperror.New(apperror.ErrValidation, "Deadline tidak boleh di masa lalu")
	}
	icon := strings.TrimSpace(input.Icon)
	if icon == "" {
		icon = "target"
	}
	color := strings.TrimSpace(input.Color)
	if color == "" {
		color = "#6366F1"
	}
	now := time.Now().UTC()
	goal := &model.Goal{
		ID:            ids.New("goal"),
		UserID:        userID,
		Name:          name,
		TargetAmount:  input.TargetAmount,
		CurrentAmount: 0,
		Deadline:      deadline,
		Icon:          icon,
		Color:         color,
		Status:        statusActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repository.Create(ctx, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

func (s *Service) Update(ctx context.Context, id, userID string, input UpdateInput) (*model.Goal, error) {
	goal, err := s.repository.Get(ctx, id, userID)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "Target tidak ditemukan")
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return nil, apperror.New(apperror.ErrValidation, "Nama target wajib diisi dan maksimal 100 karakter")
		}
		goal.Name = name
	}
	if input.TargetAmount != nil {
		if *input.TargetAmount <= 0 {
			return nil, apperror.New(apperror.ErrValidation, "Nominal target harus lebih dari 0")
		}
		goal.TargetAmount = *input.TargetAmount
	}
	if input.Deadline != nil {
		deadline := strings.TrimSpace(*input.Deadline)
		dl, err := time.Parse("2006-01-02", deadline)
		if err != nil {
			return nil, apperror.New(apperror.ErrValidation, "Format deadline harus YYYY-MM-DD")
		}
		if dl.Before(time.Now().Truncate(24 * time.Hour)) {
			return nil, apperror.New(apperror.ErrValidation, "Deadline tidak boleh di masa lalu")
		}
		goal.Deadline = deadline
	}
	if input.Icon != nil {
		goal.Icon = strings.TrimSpace(*input.Icon)
	}
	if input.Color != nil {
		goal.Color = strings.TrimSpace(*input.Color)
	}
	goal.Status = computeStatus(goal.CurrentAmount, goal.TargetAmount)
	goal.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

func (s *Service) Contribute(ctx context.Context, id, userID string, input ContributeInput) (*model.Goal, error) {
	if input.Amount == 0 {
		return nil, apperror.New(apperror.ErrValidation, "Jumlah kontribusi tidak boleh 0")
	}
	goal, err := s.repository.Get(ctx, id, userID)
	if err != nil {
		return nil, apperror.New(apperror.ErrNotFound, "Target tidak ditemukan")
	}
	newAmount := goal.CurrentAmount + input.Amount
	if newAmount < 0 {
		return nil, apperror.New(apperror.ErrValidation, "Dana terkumpul tidak boleh negatif")
	}
	if newAmount > goal.TargetAmount {
		newAmount = goal.TargetAmount
	}
	goal.CurrentAmount = newAmount
	goal.Status = computeStatus(goal.CurrentAmount, goal.TargetAmount)
	goal.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, goal); err != nil {
		return nil, err
	}
	return goal, nil
}

func (s *Service) Delete(ctx context.Context, id, userID string) error {
	if _, err := s.repository.Get(ctx, id, userID); err != nil {
		return apperror.New(apperror.ErrNotFound, "Target tidak ditemukan")
	}
	return s.repository.Delete(ctx, id, userID)
}

func computeStatus(current, target int) string {
	if current >= target {
		return statusAchieved
	}
	return statusActive
}
