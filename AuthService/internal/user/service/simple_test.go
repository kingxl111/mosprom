package service

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePasswordHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "testpassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: "verylongpasswordwithspecialchars!@#$%^&*()",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := generatePasswordHash(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.password, hash)
		})
	}
}

func TestVerifyPasswordHash(t *testing.T) {
	password := "testpassword123"
	hash, err := generatePasswordHash(password)
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "wrong password",
			password: "wrongpassword",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty hash",
			password: password,
			hash:     "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyPasswordHash(tt.password, tt.hash)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "valid length",
			length:  32,
			wantErr: false,
		},
		{
			name:    "zero length",
			length:  0,
			wantErr: false,
		},
		{
			name:    "long length",
			length:  128,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generateRandomString(tt.length)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, result, tt.length)
		})
	}

	// Test that multiple calls produce different results
	str1, err1 := generateRandomString(32)
	require.NoError(t, err1)
	str2, err2 := generateRandomString(32)
	require.NoError(t, err2)
	assert.NotEqual(t, str1, str2)
}

func TestGenerateToken(t *testing.T) {
	tests := []struct {
		name    string
		userID  int
		email   string
		wantErr bool
	}{
		{
			name:    "valid token",
			userID:  1,
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "zero user ID",
			userID:  0,
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			userID:  1,
			email:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.userID, tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, token)
		})
	}
}

func TestParseToken(t *testing.T) {
	// Generate a valid token first
	userID := 1
	email := "test@example.com"
	validToken, err := GenerateToken(userID, email)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: true,
		},
		{
			name:    "malformed token",
			token:   "not-a-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ParseToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, claims)
			assert.Equal(t, userID, claims.UserID)
			assert.Equal(t, email, claims.Email)
		})
	}
}

func TestTokenClaims(t *testing.T) {
	userID := 123
	email := "user@example.com"

	token, err := GenerateToken(userID, email)
	require.NoError(t, err)

	claims, err := ParseToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Verify claims
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
	assert.True(t, claims.ExpiresAt.Time.After(claims.IssuedAt.Time))
}

func TestTokenSigningMethod(t *testing.T) {
	userID := 1
	email := "test@example.com"

	token, err := GenerateToken(userID, email)
	require.NoError(t, err)

	// Parse token manually to check signing method
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})

	// This should fail because we're using a different secret
	assert.Error(t, err)
	// The token might be parsed but invalid due to wrong secret
	if parsedToken != nil {
		assert.False(t, parsedToken.Valid)
	}
}
