package config

import (
	"errors"
	"os"
	"strconv"
)

var _ RedisConfig = (*redisConfig)(nil)

const (
	redisHostEnvName     = "REDIS_HOST"
	redisPortEnvName     = "REDIS_PORT"
	redisPasswordEnvName = "REDIS_PASSWORD"
	redisDBEnvName       = "REDIS_DB"
)

type RedisConfig interface {
	Host() string
	Port() string
	Password() string
	DB() int
	Addr() string
}

type redisConfig struct {
	host     string
	port     string
	password string
	db       int
}

func NewRedisConfig() (RedisConfig, error) {
	host := os.Getenv(redisHostEnvName)
	if len(host) == 0 {
		return nil, errors.New("redis host not found")
	}

	port := os.Getenv(redisPortEnvName)
	if len(port) == 0 {
		return nil, errors.New("redis port not found")
	}

	password := os.Getenv(redisPasswordEnvName)

	dbStr := os.Getenv(redisDBEnvName)
	db := 0
	if dbStr != "" {
		var err error
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return nil, errors.New("invalid redis db")
		}
	}

	return &redisConfig{
		host:     host,
		port:     port,
		password: password,
		db:       db,
	}, nil
}

func (cfg *redisConfig) Host() string {
	return cfg.host
}

func (cfg *redisConfig) Port() string {
	return cfg.port
}

func (cfg *redisConfig) Password() string {
	return cfg.password
}

func (cfg *redisConfig) DB() int {
	return cfg.db
}

func (cfg *redisConfig) Addr() string {
	return cfg.host + ":" + cfg.port
}
