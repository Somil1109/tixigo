package auth

import (
	"testing"
	"time"
)

func TestAccessTokenRoundTrip(t *testing.T) {
	manager, err := NewTokenManager("a-very-long-access-secret-that-is-safe", "a-very-long-refresh-secret-that-is-safe", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	token, err := manager.NewAccessToken("user-123", RoleCustomer, time.Now())
	if err != nil {
		t.Fatalf("NewAccessToken() error = %v", err)
	}
	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken() error = %v", err)
	}
	if claims.Subject != "user-123" || claims.Role != RoleCustomer {
		t.Fatalf("claims = %+v, want user-123/customer", claims)
	}
}

func TestRefreshTokenIsStableWhenHashed(t *testing.T) {
	manager, _ := NewTokenManager("a-very-long-access-secret-that-is-safe", "a-very-long-refresh-secret-that-is-safe", 15*time.Minute, time.Hour)
	raw, hash, _, err := manager.NewRefreshToken(time.Now())
	if err != nil {
		t.Fatalf("NewRefreshToken() error = %v", err)
	}
	if hash != manager.HashRefreshToken(raw) {
		t.Fatal("refresh token hash should be reproducible")
	}
}
