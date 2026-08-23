package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"github.com/tixigo/tixigo-api/internal/config"
	"github.com/tixigo/tixigo-api/internal/database"
)

func main() {
	email := flag.String("email", "", "email of an existing Tixigo account")
	flag.Parse()
	if strings.TrimSpace(*email) == "" {
		log.Fatal("usage: go run ./cmd/promote-admin --email you@example.com")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, config.Load().DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()
	command, err := pool.Exec(ctx, `UPDATE users SET role='admin',updated_at=now() WHERE lower(email)=lower($1)`, strings.TrimSpace(*email))
	if err != nil {
		log.Fatalf("promote account: %v", err)
	}
	if command.RowsAffected() == 0 {
		log.Fatalf("no account found for %s; register it first", *email)
	}
	log.Printf("promoted %s to admin", *email)
}
