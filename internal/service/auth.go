package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"money-management-service/internal/cache"
	"money-management-service/internal/config"
	"money-management-service/internal/model"
	"money-management-service/internal/pkg/apperror"
	"money-management-service/internal/pkg/ids"
	"money-management-service/internal/repository"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type AuthService struct {
	cfg   config.Config
	store *repository.Store
	cache *cache.Cache
}

func NewAuthService(cfg config.Config, store *repository.Store, cache *cache.Cache) *AuthService {
	return &AuthService{cfg: cfg, store: store, cache: cache}
}

func (s *AuthService) Register(ctx context.Context, phone, name string, email *string, password string, referralCode *string) (*model.User, *model.UserBalance, TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BcryptCost)
	if err != nil {
		return nil, nil, TokenPair{}, err
	}

	now := time.Now().UTC()
	user := &model.User{
		ID:           ids.New("usr"),
		Phone:        phone,
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Timezone:     "Asia/Jakarta",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.store.CreateUserWithBalance(ctx, user, referralCode); err != nil {
		return nil, nil, TokenPair{}, err
	}
	balance, err := s.store.GetBalance(ctx, user.ID)
	if err != nil {
		return nil, nil, TokenPair{}, err
	}
	pair, err := s.NewTokenPair(ctx, user.ID, "user", true)
	if err != nil {
		return nil, nil, TokenPair{}, err
	}
	return user, balance, pair, nil
}

func (s *AuthService) Login(ctx context.Context, phone, password string) (*model.User, *model.UserBalance, TokenPair, error) {
	user, err := s.store.GetUserByPhone(ctx, phone)
	if err != nil {
		return nil, nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Nomor telepon atau password salah")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Nomor telepon atau password salah")
	}
	if !user.IsActive {
		return nil, nil, TokenPair{}, apperror.New(apperror.ErrForbidden, "Akun dinonaktifkan. Hubungi admin.")
	}
	balance, err := s.store.GetBalance(ctx, user.ID)
	if err != nil {
		return nil, nil, TokenPair{}, err
	}
	pair, err := s.NewTokenPair(ctx, user.ID, "user", true)
	if err != nil {
		return nil, nil, TokenPair{}, err
	}
	_ = s.store.PruneRefreshTokens(ctx, user.ID, 5)
	s.cache.SetJSON(ctx, "user:"+user.ID, map[string]interface{}{"user": user, "balance": balance}, 15*time.Minute)
	return user, balance, pair, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.ParseToken(refreshToken, true)
	if err != nil || claims.Type != "refresh" {
		return TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Refresh token expired atau tidak valid")
	}
	hash := HashToken(refreshToken)
	stored, err := s.store.GetRefreshTokenByHash(ctx, hash)
	if err != nil || stored.ExpiresAt.Before(time.Now()) {
		return TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Refresh token expired atau tidak valid")
	}
	_ = s.store.DeleteRefreshTokenByHash(ctx, hash)
	return s.NewTokenPair(ctx, claims.Subject, claims.Role, true)
}

func (s *AuthService) Logout(ctx context.Context, userID, refreshToken string) error {
	if refreshToken != "" {
		_ = s.store.DeleteRefreshTokenByHash(ctx, HashToken(refreshToken))
	}
	s.cache.Delete(ctx, "user:"+userID)
	return nil
}

func (s *AuthService) AdminLogin(ctx context.Context, username, password string) (*model.Admin, TokenPair, error) {
	admin, err := s.store.GetAdminByUsername(ctx, username)
	if err != nil {
		return nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Username atau password salah")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Username atau password salah")
	}
	pair, err := s.NewTokenPair(ctx, admin.ID, admin.Role, false)
	return admin, pair, err
}

func (s *AuthService) NewTokenPair(ctx context.Context, subject, role string, persistRefresh bool) (TokenPair, error) {
	access, err := s.sign(subject, role, "access", s.cfg.JWTAccessExpiry, false)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := s.sign(subject, role, "refresh", s.cfg.JWTRefreshExpiry, true)
	if err != nil {
		return TokenPair{}, err
	}
	if persistRefresh {
		err = s.store.CreateRefreshToken(ctx, model.RefreshToken{
			ID:        ids.New("rft"),
			UserID:    subject,
			TokenHash: HashToken(refresh),
			ExpiresAt: time.Now().Add(s.cfg.JWTRefreshExpiry),
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return TokenPair{}, err
		}
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.cfg.JWTAccessExpiry.Seconds()),
	}, nil
}

func (s *AuthService) sign(subject, role, typ string, ttl time.Duration, refresh bool) (string, error) {
	secret := s.cfg.JWTAccessSecret
	if refresh {
		secret = s.cfg.JWTRefreshSecret
	}
	now := time.Now()
	claims := AppClaims{
		Role: role,
		Type: typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ID:        ids.RandomHex(8),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *AuthService) ParseToken(tokenValue string, refresh bool) (*AppClaims, error) {
	secret := s.cfg.JWTAccessSecret
	if refresh {
		secret = s.cfg.JWTRefreshSecret
	}
	token, err := jwt.ParseWithClaims(tokenValue, &AppClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperror.ErrUnauthorized
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, apperror.ErrUnauthorized
	}
	claims, ok := token.Claims.(*AppClaims)
	if !ok {
		return nil, apperror.ErrUnauthorized
	}
	if claims.Subject == "" || claims.Role == "" {
		return nil, apperror.ErrUnauthorized
	}
	return claims, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return apperror.New(apperror.ErrUnauthorized, "Password lama salah")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.cfg.BcryptCost)
	if err != nil {
		return err
	}
	s.cache.Delete(ctx, "user:"+userID)
	return s.store.UpdatePassword(ctx, userID, string(hash))
}

func (s *AuthService) SeedAdmin(ctx context.Context) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.AdminDefaultPassword), s.cfg.BcryptCost)
	if err != nil {
		return err
	}
	return s.store.CreateAdmin(ctx, model.Admin{
		ID:           ids.New("adm"),
		Username:     s.cfg.AdminDefaultUsername,
		PasswordHash: string(hash),
		Role:         "superadmin",
		CreatedAt:    time.Now().UTC(),
	})
}

type AppClaims struct {
	Role string `json:"role"`
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func IsAppError(err error, target error) bool {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return errors.Is(appErr.Err, target)
	}
	return errors.Is(err, target)
}
