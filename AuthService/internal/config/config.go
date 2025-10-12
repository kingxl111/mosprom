package config

import (
	"os"

	"github.com/joho/godotenv"
)

func Load(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}

	return nil
}

func GetJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// В продакшене это должно быть ошибкой
		return "default-secret-key-change-in-production"
	}
	return secret
}
