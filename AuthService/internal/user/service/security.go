package service

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/go-faster/errors"

	"github.com/golang-jwt/jwt/v5"
)

const (
	salt       = "kqwemjksdnfhaksrmksvj283njwksdf"
	signingKey = "821nci1nc1234ubcz,mszd2jcv1wd23"
	tokenTTL   = time.Minute * 1000
)

type tokenClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
}

func generatePasswordHash(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	return fmt.Sprintf("%x", hash.Sum([]byte(salt)))
}

func GenerateToken(username string) (string, error) {
	claims := &tokenClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(signingKey))
}

func ParseToken(accessToken string) (string, error) {
	token, err := jwt.ParseWithClaims(accessToken, &tokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(signingKey), nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(*tokenClaims)
	if !ok || claims == nil {
		return "", errors.New("token claims are not of type *tokenClaims")
	}

	if claims.ExpiresAt.Before(time.Now()) {
		return "", errors.New("token expired")
	}

	return claims.Username, nil
}
