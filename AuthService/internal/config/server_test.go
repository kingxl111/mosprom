package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHTTPConfig(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func()
		wantErr  bool
	}{
		{
			name: "valid config",
			setupEnv: func() {
				os.Setenv("HTTP_HOST", "localhost")
				os.Setenv("HTTP_PORT", "8080")
			},
			wantErr: false,
		},
		{
			name: "missing host",
			setupEnv: func() {
				os.Unsetenv("HTTP_HOST")
				os.Setenv("HTTP_PORT", "8080")
			},
			wantErr: true,
		},
		{
			name: "missing port",
			setupEnv: func() {
				os.Setenv("HTTP_HOST", "localhost")
				os.Unsetenv("HTTP_PORT")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer func() {
				os.Unsetenv("HTTP_HOST")
				os.Unsetenv("HTTP_PORT")
			}()

			cfg, err := NewHTTPConfig()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, "localhost:8080", cfg.Address())
			}
		})
	}
}

func TestHTTPConfig_Address(t *testing.T) {
	cfg := &httpConfig{
		host: "example.com",
		port: "9090",
	}

	result := cfg.Address()
	assert.Equal(t, "example.com:9090", result)
}
