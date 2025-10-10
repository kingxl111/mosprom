package user

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrInvalidToken       = errors.New("invalid token")

	ErrUserExists   = errors.New("user already exists")
	ErrUserInactive = errors.New("user is inactive")
	ErrUserNotFound = errors.New("user not found")

	ErrSessionNotFound   = errors.New("session not found")
	ErrRefreshTokenEmpty = errors.New("refresh token is empty")

	ErrInternal = errors.New("internal error")
)
