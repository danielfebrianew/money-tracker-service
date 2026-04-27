package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"money-management-service/internal/config"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
)

type ReferralService struct {
	cfg   config.Config
	store *repository.Store
}

func NewReferralService(cfg config.Config, store *repository.Store) *ReferralService {
	return &ReferralService{cfg: cfg, store: store}
}

func (s *ReferralService) Summary(ctx context.Context, userID string) (map[string]interface{}, error) {
	code, err := s.store.GetReferralCodeByUser(ctx, userID)
	if err != nil {
		return map[string]interface{}{
			"code":                nil,
			"total_referrals":     0,
			"active_referrals":    0,
			"total_earned":        0,
			"pending_payout":      0,
			"commission_per_user": 5000,
		}, nil
	}
	total, active, earned, pending, err := s.store.ReferralSummary(ctx, code.Code)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"code":                code.Code,
		"total_referrals":     total,
		"active_referrals":    active,
		"total_earned":        earned,
		"pending_payout":      pending,
		"commission_per_user": code.Commission,
	}, nil
}

func (s *ReferralService) Generate(ctx context.Context, userID string) (map[string]interface{}, error) {
	existing, err := s.store.GetReferralCodeByUser(ctx, userID)
	if err == nil {
		return referralResponse(s.cfg.AppURL, existing.Code), nil
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	prefix := "USR"
	name := strings.ToUpper(strings.TrimSpace(user.Name))
	if len(name) >= 3 {
		prefix = name[:3]
	}
	prefix = strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r
		}
		return -1
	}, prefix)
	if prefix == "" {
		prefix = "USR"
	}
	codeValue := fmt.Sprintf("%s%s", prefix, ids.RandomHex(2)[:3])
	code := model.ReferralCode{
		ID:         ids.New("ref"),
		UserID:     &user.ID,
		Code:       codeValue,
		Name:       user.Name,
		Phone:      &user.Phone,
		Commission: 5000,
		IsActive:   true,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.store.CreateReferralCode(ctx, code); err != nil {
		return nil, err
	}
	return referralResponse(s.cfg.AppURL, code.Code), nil
}

func referralResponse(appURL, code string) map[string]interface{} {
	return map[string]interface{}{
		"code":          code,
		"referral_link": strings.TrimRight(appURL, "/") + "/ref/" + code,
	}
}
