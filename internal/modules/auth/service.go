package auth

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
)

type Service struct {
	cfg        config.Config
	repository *Repository
	cache      *cache.Cache
}

func NewService(cfg config.Config, repository *Repository, cache *cache.Cache) *Service {
	return &Service{cfg: cfg, repository: repository, cache: cache}
}

func (s *Service) Register(ctx context.Context, phone, name string, email *string, password string, referralCode *string) (TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cfg.BcryptCost)
	if err != nil {
		return TokenPair{}, err
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
	if err := s.repository.CreateUserWithBalance(ctx, user, referralCode); err != nil {
		return TokenPair{}, err
	}
	pair, err := s.NewUserTokenPair(ctx, user.ID, "user")
	if err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

func (s *Service) Login(ctx context.Context, identifier, password string) (*model.User, *model.UserBalance, TokenPair, error) {
	var user *model.User
	var err error
	if phonePattern.MatchString(identifier) {
		user, err = s.repository.GetUserByPhone(ctx, identifier)
	} else {
		user, err = s.repository.GetUserByEmail(ctx, identifier)
	}
	if err != nil {
		return nil, nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Email/nomor telepon atau password salah")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Email/nomor telepon atau password salah")
	}
	if !user.IsActive {
		return nil, nil, TokenPair{}, apperror.New(apperror.ErrForbidden, "Akun dinonaktifkan. Hubungi admin.")
	}
	balance, err := s.repository.GetBalance(ctx, user.ID)
	if err != nil {
		return nil, nil, TokenPair{}, err
	}
	pair, err := s.NewUserTokenPair(ctx, user.ID, "user")
	if err != nil {
		return nil, nil, TokenPair{}, err
	}
	_ = s.repository.PruneRefreshTokens(ctx, user.ID, 5)
	s.cache.SetJSON(ctx, "user:"+user.ID, map[string]interface{}{"user": user, "balance": balance}, 15*time.Minute)
	return user, balance, pair, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.ParseToken(refreshToken, true)
	if err != nil || claims.Type != "refresh" {
		return TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Refresh token expired atau tidak valid")
	}
	hash := HashToken(refreshToken)
	stored, err := s.repository.GetRefreshTokenByHash(ctx, hash)
	if err != nil || stored.ExpiresAt.Before(time.Now()) {
		return TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Refresh token expired atau tidak valid")
	}
	_ = s.repository.DeleteRefreshTokenByHash(ctx, hash)
	return s.NewUserTokenPair(ctx, claims.Subject, claims.Role)
}

func (s *Service) Logout(ctx context.Context, userID, refreshToken string) error {
	if refreshToken != "" {
		_ = s.repository.DeleteRefreshTokenByHash(ctx, HashToken(refreshToken))
	}
	s.cache.Delete(ctx, "user:"+userID)
	return nil
}

func (s *Service) AdminLogin(ctx context.Context, username, password string) (*model.Admin, TokenPair, error) {
	admin, err := s.repository.GetAdminByUsername(ctx, username)
	if err != nil {
		return nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Username atau password salah")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)); err != nil {
		return nil, TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Username atau password salah")
	}
	pair, err := s.NewAdminTokenPair(ctx, admin.ID, admin.Role)
	if err != nil {
		return nil, TokenPair{}, err
	}
	_ = s.repository.PruneAdminRefreshTokens(ctx, admin.ID, 5)
	return admin, pair, nil
}

func (s *Service) AdminRefresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.ParseToken(refreshToken, true)
	if err != nil || claims.Type != "refresh" {
		return TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Refresh token expired atau tidak valid")
	}
	if claims.Role != "admin" && claims.Role != "superadmin" {
		return TokenPair{}, apperror.New(apperror.ErrForbidden, "Token bukan untuk admin")
	}
	hash := HashToken(refreshToken)
	stored, err := s.repository.GetAdminRefreshTokenByHash(ctx, hash)
	if err != nil || stored.ExpiresAt.Before(time.Now()) {
		return TokenPair{}, apperror.New(apperror.ErrUnauthorized, "Refresh token expired atau tidak valid")
	}
	_ = s.repository.DeleteAdminRefreshTokenByHash(ctx, hash)
	return s.NewAdminTokenPair(ctx, claims.Subject, claims.Role)
}

func (s *Service) AdminLogout(ctx context.Context, refreshToken string) error {
	if refreshToken != "" {
		_ = s.repository.DeleteAdminRefreshTokenByHash(ctx, HashToken(refreshToken))
	}
	return nil
}

// AudienceUser/AudienceAdmin are passed to newTokenPair to pick the correct
// refresh-token store. Zero value (empty string) means "don't persist".
const (
	audienceUser  = "user"
	audienceAdmin = "admin"
)

// NewUserTokenPair issues an access+refresh pair for a regular user. Refresh is
// always persisted in refresh_tokens (FK to users).
func (s *Service) NewUserTokenPair(ctx context.Context, subject, role string) (TokenPair, error) {
	return s.newTokenPair(ctx, subject, role, audienceUser)
}

// NewAdminTokenPair issues an access+refresh pair for an admin. Refresh is
// always persisted in admin_refresh_tokens (FK to admins).
func (s *Service) NewAdminTokenPair(ctx context.Context, subject, role string) (TokenPair, error) {
	return s.newTokenPair(ctx, subject, role, audienceAdmin)
}

func (s *Service) newTokenPair(ctx context.Context, subject, role, audience string) (TokenPair, error) {
	access, err := s.sign(subject, role, "access", s.cfg.JWTAccessExpiry, false)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, err := s.sign(subject, role, "refresh", s.cfg.JWTRefreshExpiry, true)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.persistRefreshToken(ctx, subject, HashToken(refresh), audience); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(s.cfg.JWTAccessExpiry.Seconds()),
	}, nil
}

func (s *Service) persistRefreshToken(ctx context.Context, subject, hash, audience string) error {
	now := time.Now().UTC()
	expires := time.Now().Add(s.cfg.JWTRefreshExpiry)
	switch audience {
	case audienceAdmin:
		return s.repository.CreateAdminRefreshToken(ctx, model.AdminRefreshToken{
			ID:        ids.New("art"),
			AdminID:   subject,
			TokenHash: hash,
			ExpiresAt: expires,
			CreatedAt: now,
		})
	default:
		return s.repository.CreateRefreshToken(ctx, model.RefreshToken{
			ID:        ids.New("rft"),
			UserID:    subject,
			TokenHash: hash,
			ExpiresAt: expires,
			CreatedAt: now,
		})
	}
}

func (s *Service) sign(subject, role, typ string, ttl time.Duration, refresh bool) (string, error) {
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

func (s *Service) ParseToken(tokenValue string, refresh bool) (*AppClaims, error) {
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

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.repository.GetUserByID(ctx, userID)
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
	return s.repository.UpdatePassword(ctx, userID, string(hash))
}

func (s *Service) SeedAdmin(ctx context.Context) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.AdminDefaultPassword), s.cfg.BcryptCost)
	if err != nil {
		return err
	}
	return s.repository.CreateAdmin(ctx, model.Admin{
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
