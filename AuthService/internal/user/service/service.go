package service

import (
	"context"
	"fmt"
	repo "github.com/kingxl111/mosprom/AuthService/internal/repository"
	pg "github.com/kingxl111/mosprom/AuthService/internal/repository/postgres"
	models "github.com/kingxl111/mosprom/AuthService/internal/user"
	"time"

	"github.com/go-faster/errors"
	"github.com/redis/go-redis/v9"
)

type AuthService struct {
	repo  Repository
	redis *redis.Client
}

func NewAuthService(repo Repository, redis *redis.Client) *AuthService {
	return &AuthService{
		repo:  repo,
		redis: redis,
	}
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Проверяем, что пользователя нет
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, repo.ErrorUserNotFound) {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if user != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	passwordHash := generatePasswordHash(req.Password)
	newUser := &pg.User{
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, newUser); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	access, err := GenerateToken(newUser.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh := generateRandomString(64)
	token := &pg.RefreshToken{
		UserID:    newUser.ID,
		Token:     refresh,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.repo.SaveRefreshToken(ctx, token); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	if s.redis != nil {
		s.redis.Set(ctx, fmt.Sprintf("refresh:%s", refresh), newUser.ID, 24*time.Hour)
	}

	return &models.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repo.ErrorUserNotFound) {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user is inactive")
	}

	if user.PasswordHash != generatePasswordHash(req.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	access, err := GenerateToken(user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh := generateRandomString(64)
	refreshToken := &pg.RefreshToken{
		UserID:    user.ID,
		Token:     refresh,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.repo.SaveRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	session := &pg.UserSession{
		UserID:       user.ID,
		IPAddress:    req.IP,
		UserAgent:    req.UserAgent,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		// Не критическая ошибка — просто логируем
		fmt.Printf("create session failed: %v\n", err)
	}

	if s.redis != nil {
		s.redis.Set(ctx, fmt.Sprintf("refresh:%s", refresh), user.ID, 24*time.Hour)
	}

	return &models.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*models.TokenResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("empty refresh token")
	}

	// Redis-кэш (опционально) — не заменяет проверку в БД
	if s.redis != nil {
		val, err := s.redis.Get(ctx, fmt.Sprintf("refresh:%s", refreshToken)).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("redis get: %w", err)
		}
		// если val == "" и err == nil — скорее всего некорректный токен, но всё равно проверим БД
	}

	rt, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, repo.ErrorNotFound) {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	if rt.Revoked {
		return nil, fmt.Errorf("refresh token revoked")
	}
	if rt.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("refresh token expired")
	}

	user, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrorNotFound) {
			return nil, fmt.Errorf("user for refresh token not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	access, err := GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Optionally rotate refresh token:
	// — можно (и рекомендуется) создавать новый refresh token, сохранить и аннулировать старый.
	// Ниже — пример без ротации (если хочешь ротацию — скажу как вставить).
	return &models.TokenResponse{
		AccessToken: access,
	}, nil
}

// GetUserByID просто обёртка для handler'ов
func (s *AuthService) GetUserByID(ctx context.Context, id int) (*models.UserResponse, error) {
	u, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrorNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return &models.UserResponse{
		ID:        fmt.Sprintf("%d", u.ID),
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return fmt.Errorf("empty token")
	}

	if err := s.repo.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if s.redis != nil {
		s.redis.Del(ctx, fmt.Sprintf("refresh:%s", refreshToken))
	}

	return nil
}
