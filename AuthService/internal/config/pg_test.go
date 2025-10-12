package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPGConfig(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func()
		wantErr  bool
	}{
		{
			name: "valid config with individual vars",
			setupEnv: func() {
				os.Setenv("DATABASE_HOST", "localhost")
				os.Setenv("DATABASE_PORT", "5432")
				os.Setenv("DATABASE_NAME", "testdb")
				os.Setenv("DATABASE_USER", "testuser")
				os.Setenv("DATABASE_PASSWORD", "testpass")
			},
			wantErr: false,
		},
		{
			name: "valid config with DSN",
			setupEnv: func() {
				os.Setenv("PG_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer func() {
				os.Unsetenv("DATABASE_HOST")
				os.Unsetenv("DATABASE_PORT")
				os.Unsetenv("DATABASE_NAME")
				os.Unsetenv("DATABASE_USER")
				os.Unsetenv("DATABASE_PASSWORD")
				os.Unsetenv("PG_DSN")
			}()

			cfg, err := NewPGConfig()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}
