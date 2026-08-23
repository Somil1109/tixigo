package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
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
	"github.com/tixigo/tixigo-api/internal/realtime"
	"github.com/tixigo/tixigo-api/internal/seat"
	"github.com/tixigo/tixigo-api/internal/waitlist"
)

func main() {
	cfg := config.Load()
	if cfg.Environment == "production" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	}
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
	hub := realtime.NewHub()
	email := notification.NewEmailSender(cfg)
	waiting := waitlist.NewStore(pool)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		store := seat.NewStore(pool)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				screeningIDs, err := store.ReleaseExpired(ctx)
				if err != nil {
					log.Printf("release expired seat holds: %v", err)
					continue
				}
				for _, screeningID := range screeningIDs {
					hub.Publish(screeningID)
					offers, matchErr := waiting.Match(ctx, screeningID)
					if matchErr != nil {
						log.Printf("match waitlist for screening %s: %v", screeningID, matchErr)
						continue
					}
					waitlist.NotifyOffers(ctx, email, offers)
					if len(offers) > 0 {
						hub.Publish(screeningID)
					}
				}
			}
		}
	}()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(cfg, pool, tokens, email, hub),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
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
