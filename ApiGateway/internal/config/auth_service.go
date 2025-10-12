package config

import (
	"errors"
	"os"
)

var _ AuthServiceConfig = (*authServiceConfig)(nil)

const (
	authServiceHostEnvName = "AUTH_SERVICE_HOST"
	authServicePortEnvName = "AUTH_SERVICE_PORT"
)

type AuthServiceConfig interface {
	Host() string
	Port() string
	BaseURL() string
}

type authServiceConfig struct {
	host string
	port string
}

func NewAuthServiceConfig() (AuthServiceConfig, error) {
	host := os.Getenv(authServiceHostEnvName)
	if len(host) == 0 {
		return nil, errors.New("auth service host not found")
	}

	port := os.Getenv(authServicePortEnvName)
	if len(port) == 0 {
		return nil, errors.New("auth service port not found")
	}

	return &authServiceConfig{
		host: host,
		port: port,
	}, nil
}

func (cfg *authServiceConfig) Host() string {
	return cfg.host
}

func (cfg *authServiceConfig) Port() string {
	return cfg.port
}

func (cfg *authServiceConfig) BaseURL() string {
	return "http://" + cfg.host + ":" + cfg.port
}
