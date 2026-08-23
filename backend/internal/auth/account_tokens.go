package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountTokenStore struct{ pool *pgxpool.Pool }

func NewAccountTokenStore(pool *pgxpool.Pool) *AccountTokenStore { return &AccountTokenStore{pool} }

func NewAccountToken() (raw, hash string, err error) {
	value := make([]byte, 32)
	if _, err = rand.Read(value); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(value)
	digest := sha256.Sum256([]byte(raw))
	return raw, base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func HashAccountToken(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func (s *AccountTokenStore) Create(ctx context.Context, userID, purpose, hash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO account_tokens (user_id,purpose,token_hash,expires_at) VALUES ($1,$2,$3,$4)`, userID, purpose, hash, expiresAt)
	return err
}

func (s *AccountTokenStore) VerifyEmail(ctx context.Context, hash string) error {
	result, err := s.pool.Exec(ctx, `WITH token AS (UPDATE account_tokens SET consumed_at=now() WHERE token_hash=$1 AND purpose='verify_email' AND consumed_at IS NULL AND expires_at>now() RETURNING user_id) UPDATE users SET email_verified_at=COALESCE(email_verified_at,now()),updated_at=now() WHERE id=(SELECT user_id FROM token)`, hash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInvalidAccountToken
	}
	return nil
}
