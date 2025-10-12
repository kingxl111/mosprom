package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantErr  bool
		setupEnv func()
	}{
		{
			name:    "load existing env file",
			path:    ".env.test",
			wantErr: false,
			setupEnv: func() {
				// Create a test .env file
				content := "TEST_VAR=test_value\nANOTHER_VAR=another_value\n"
				err := os.WriteFile(".env.test", []byte(content), 0644)
				require.NoError(t, err)
			},
		},
		{
			name:     "load non-existing env file",
			path:     ".env.nonexistent",
			wantErr:  true,
			setupEnv: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer func() {
				// Clean up test file
				os.Remove(".env.test")
			}()

			err := Load(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetJWTSecret(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
		setupEnv func()
	}{
		{
			name:     "JWT_SECRET set",
			envValue: "my-secret-key",
			expected: "my-secret-key",
			setupEnv: func() {
				os.Setenv("JWT_SECRET", "my-secret-key")
			},
		},
		{
			name:     "JWT_SECRET not set",
			envValue: "",
			expected: "default-secret-key-change-in-production",
			setupEnv: func() {
				os.Unsetenv("JWT_SECRET")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer func() {
				os.Unsetenv("JWT_SECRET")
			}()

			result := GetJWTSecret()
			assert.Equal(t, tt.expected, result)
		})
	}
}
