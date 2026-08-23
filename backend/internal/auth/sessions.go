package auth

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type SessionStore struct{ pool *pgxpool.Pool }

func NewSessionStore(pool *pgxpool.Pool) *SessionStore { return &SessionStore{pool} }
func (s *SessionStore) Create(ctx context.Context, userID, hash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO refresh_tokens (user_id,token_hash,expires_at) VALUES ($1,$2,$3)`, userID, hash, expiresAt)
	return err
}
func (s *SessionStore) UserID(ctx context.Context, hash string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT user_id::text FROM refresh_tokens WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now()`, hash).Scan(&id)
	return id, err
}
func (s *SessionStore) Revoke(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`, hash)
	return err
}
