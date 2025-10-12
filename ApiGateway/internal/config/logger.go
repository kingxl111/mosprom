package config

import (
	"errors"
	"log/slog"
	"os"
	"strings"
)

var _ LoggerConfig = (*loggerConfig)(nil)

const (
	logLevelEnvName = "LOG_LEVEL"
)

type LoggerConfig interface {
	Level() slog.Level
}

type loggerConfig struct {
	level slog.Level
}

func NewLoggerConfig() (LoggerConfig, error) {
	levelStr := os.Getenv(logLevelEnvName)
	if levelStr == "" {
		levelStr = "INFO"
	}

	var level slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN", "WARNING":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		return nil, errors.New("invalid log level")
	}

	return &loggerConfig{
		level: level,
	}, nil
}

func (cfg *loggerConfig) Level() slog.Level {
	return cfg.level
}
