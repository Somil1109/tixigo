package auth

import "testing"

func TestAccountTokenHash(t *testing.T) {
	raw, hash, err := NewAccountToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == "" {
		t.Fatal("token and hash must not be empty")
	}
	if got := HashAccountToken(raw); got != hash {
		t.Fatalf("hash = %q, want %q", got, hash)
	}
}
