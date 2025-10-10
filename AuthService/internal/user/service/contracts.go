package service

import (
	"context"
	"github.com/kingxl111/mosprom/AuthService/internal/repository/postgres"
)

//go:generate mockgen -source=contracts.go -destination=mocks.go -package=service
type (
	UserRepository interface {
		CreateUser(ctx context.Context, user *postgres.User) error
		GetUserByEmail(ctx context.Context, email string) (*postgres.User, error)
		GetUserByID(ctx context.Context, id int) (*postgres.User, error) // <- added
		UpdateLastLogin(ctx context.Context, userID int) error
	}
	TokenRepository interface {
		SaveRefreshToken(ctx context.Context, token *postgres.RefreshToken) error
		GetRefreshToken(ctx context.Context, token string) (*postgres.RefreshToken, error) // <- added
		RevokeRefreshToken(ctx context.Context, token string) error
	}
	SessionRepository interface {
		CreateSession(ctx context.Context, s *postgres.UserSession) error
		UpdateSessionActivity(ctx context.Context, sessionID int) error
	}
)

type Repository interface {
	UserRepository
	TokenRepository
	SessionRepository
}
