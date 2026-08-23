package auth

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID, Email, FullName string
	Role                Role
	EmailVerifiedAt     *time.Time
}
type UserStore struct{ pool *pgxpool.Pool }

func NewUserStore(pool *pgxpool.Pool) *UserStore { return &UserStore{pool} }
func (s *UserStore) Create(ctx context.Context, email, name, hash string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `INSERT INTO users (email, full_name, password_hash) VALUES ($1,$2,$3) RETURNING id::text,email,full_name,role`, strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(name), hash).Scan(&u.ID, &u.Email, &u.FullName, &u.Role)
	return u, err
}
func (s *UserStore) ByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT id::text,email,full_name,role,email_verified_at,password_hash FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(email))).Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.EmailVerifiedAt, &hash)
	if err == pgx.ErrNoRows {
		return User{}, "", ErrInvalidCredentials
	}
	return u, hash, err
}
func (s *UserStore) ByID(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `SELECT id::text,email,full_name,role,email_verified_at FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.EmailVerifiedAt)
	return u, err
}

var ErrInvalidCredentials = pgx.ErrNoRows
