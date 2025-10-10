package service

import (
	"context"
	"fmt"
	repo "github.com/kingxl111/mosprom/AuthService/internal/repository/postgres"
	"time"

	"github.com/go-faster/errors"
	"github.com/redis/go-redis/v9"
)

// AuthService описывает основной сервис авторизации
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

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Проверяем, что пользователя нет
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, repo.ErrNotFound) {
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	if user != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	passwordHash := generatePasswordHash(req.Password)
	newUser := &postgres.User{
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

	access, err := GenerateAccessToken(newUser.ID, newUser.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh := generateRandomString(64)
	token := &postgres.RefreshToken{
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

	return &AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
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

	access, err := GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refresh := generateRandomString(64)
	refreshToken := &postgres.RefreshToken{
		UserID:    user.ID,
		Token:     refresh,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.repo.SaveRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("save refresh token: %w", err)
	}

	session := &postgres.UserSession{
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

	return &AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("empty refresh token")
	}

	// Проверка в Redis (если используется)
	if s.redis != nil {
		val, err := s.redis.Get(ctx, fmt.Sprintf("refresh:%s", refreshToken)).Result()
		if err == redis.Nil {
			// нет в кэше → проверяем в БД
		} else if err != nil {
			return nil, fmt.Errorf("redis get: %w", err)
		} else if val == "" {
			return nil, fmt.Errorf("invalid or expired refresh token")
		}
	}

	rt, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, fmt.Errorf("invalid refresh token")
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}

	if rt.Revoked || rt.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("refresh token expired or revoked")
	}

	user, err := s.repo.GetUserByEmail(ctx, rt.User.Email)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	access, err := GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	return &TokenResponse{
		AccessToken: access,
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
