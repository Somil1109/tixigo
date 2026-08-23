package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const connectionTimeout = 10 * time.Second

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	connectCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	pool, err := pgxpool.New(connectCtx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return pool, nil
}
