package config

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLoggerConfig(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantErr  bool
	}{
		{
			name:     "valid debug level",
			envValue: "debug",
			wantErr:  false,
		},
		{
			name:     "valid info level",
			envValue: "info",
			wantErr:  false,
		},
		{
			name:     "valid warn level",
			envValue: "warn",
			wantErr:  false,
		},
		{
			name:     "valid error level",
			envValue: "error",
			wantErr:  false,
		},
		{
			name:     "invalid level",
			envValue: "invalid",
			wantErr:  true,
		},
		{
			name:     "empty level",
			envValue: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOGGER_LEVEL", tt.envValue)
			defer os.Unsetenv("LOGGER_LEVEL")

			cfg, err := NewLoggerConfig()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				expectedLevel := tt.envValue
				if expectedLevel == "debug" {
					expectedLevel = "DEBUG"
				} else if expectedLevel == "info" {
					expectedLevel = "INFO"
				} else if expectedLevel == "warn" {
					expectedLevel = "WARN"
				} else if expectedLevel == "error" {
					expectedLevel = "ERROR"
				}
				assert.Equal(t, expectedLevel, cfg.Level().String())
			}
		})
	}
}

func TestLoggerConfig_Level(t *testing.T) {
	cfg := &loggerConfig{
		level: slog.LevelDebug,
	}

	result := cfg.Level()
	assert.Equal(t, slog.LevelDebug, result)
}
