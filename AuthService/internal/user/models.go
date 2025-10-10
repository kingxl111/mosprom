package user

import "time"

type User struct {
	ID        int
	Email     string
	Role      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	LastLogin *time.Time
}

type Credentials struct {
	Email    string
	Password string
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}
