package http_server

import (
	"context"

	service "github.com/kingxl111/mosprom/AuthService/internal/user"
)

//go:generate mockgen -source=contracts.go -destination=mocks.go -package=http_server
type AuthService interface {
	Register(ctx context.Context, req *service.RegisterRequest) (*service.AuthResponse, error)
	Login(ctx context.Context, req *service.LoginRequest) (*service.AuthResponse, error)
	Refresh(ctx context.Context, token string) (*service.TokenResponse, error)
	Logout(ctx context.Context, token string) error
}
