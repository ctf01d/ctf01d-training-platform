package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	jwt.RegisteredClaims
	Role     string `json:"role"`
	UserName string `json:"user_name"`
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttlHours int) *Manager {
	return &Manager{
		secret: []byte(secret),
		ttl:    time.Duration(ttlHours) * time.Hour,
	}
}

func (m *Manager) Generate(userID int64, role, userName string) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   int64ToString(userID),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Role:     role,
		UserName: userName,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func int64ToString(n int64) string {
	return fmt.Sprintf("%d", n)
}
