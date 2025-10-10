package postgres

import "time"

type User struct {
	ID           int
	Email        string
	PasswordHash string
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLogin    *time.Time
}

type RefreshToken struct {
	ID        int
	UserID    int
	Token     string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

type UserSession struct {
	ID           int
	UserID       int
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
	LastActivity time.Time
}
