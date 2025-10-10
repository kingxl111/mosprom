package service

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/go-faster/errors"
	"github.com/golang-jwt/jwt/v5"
)

const (
	salt       = "kqwemjksdnfhaksrmksvj283njwksdf"
	signingKey = "821nci1nc1234ubcz,mszd2jcv1wd23"
	tokenTTL   = time.Minute * 60 // 1 hour
)

type tokenClaims struct {
	jwt.RegisteredClaims
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
}

func generatePasswordHash(password string) string {
	h := sha256.New()
	h.Write([]byte(password))
	sum := h.Sum([]byte(salt))
	return hex.EncodeToString(sum)
}

func GenerateToken(userID int, email string) (string, error) {
	claims := &tokenClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(signingKey))
}

func ParseToken(accessToken string) (*tokenClaims, error) {
	token, err := jwt.ParseWithClaims(accessToken, &tokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(signingKey), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*tokenClaims)
	if !ok || claims == nil {
		return nil, errors.New("invalid token claims")
	}

	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}
