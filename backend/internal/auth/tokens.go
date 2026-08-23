package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleCustomer  Role = "customer"
	RoleOrganiser Role = "organiser"
	RoleAdmin     Role = "admin"
)

type AccessClaims struct {
	Role Role `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewTokenManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) (*TokenManager, error) {
	if len(accessSecret) < 32 || len(refreshSecret) < 32 {
		return nil, errors.New("JWT secrets must each be at least 32 characters")
	}
	return &TokenManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}, nil
}

func (manager *TokenManager) NewAccessToken(userID string, role Role, now time.Time) (string, error) {
	claims := AccessClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(manager.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(manager.accessSecret)
}

func (manager *TokenManager) ParseAccessToken(token string) (AccessClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return manager.accessSecret, nil
	})
	if err != nil {
		return AccessClaims{}, fmt.Errorf("parse access token: %w", err)
	}
	claims, ok := parsed.Claims.(*AccessClaims)
	if !ok || !parsed.Valid {
		return AccessClaims{}, errors.New("invalid access token")
	}
	return *claims, nil
}

func (manager *TokenManager) NewRefreshToken(now time.Time) (raw string, hash string, expiresAt time.Time, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(bytes)
	return raw, manager.HashRefreshToken(raw), now.Add(manager.refreshTTL), nil
}

func (manager *TokenManager) HashRefreshToken(raw string) string {
	mac := hmac.New(sha256.New, manager.refreshSecret)
	_, _ = mac.Write([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
