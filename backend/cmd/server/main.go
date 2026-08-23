package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tixigo/tixigo-api/internal/auth"
	"github.com/tixigo/tixigo-api/internal/config"
	"github.com/tixigo/tixigo-api/internal/database"
	"github.com/tixigo/tixigo-api/internal/httpapi"
	"github.com/tixigo/tixigo-api/internal/notification"
	"github.com/tixigo/tixigo-api/internal/seat"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database unavailable: %v", err)
	}
	defer pool.Close()
	tokens, err := auth.NewTokenManager(cfg.AccessTokenSecret, cfg.RefreshTokenSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		log.Fatalf("authentication configuration: %v", err)
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		store := seat.NewStore(pool)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := store.ReleaseExpired(ctx); err != nil {
					log.Printf("release expired seat holds: %v", err)
				}
			}
		}
	}()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(cfg, pool, tokens, notification.NewEmailSender(cfg)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Tixigo API listening on :%s (%s)", cfg.Port, cfg.Environment)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve HTTP: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown HTTP server: %v", err)
	}
}
