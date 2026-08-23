package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment        string
	Port               string
	WebOrigin          string
	DatabaseURL        string
	AccessTokenSecret  string
	RefreshTokenSecret string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	EmailFrom          string
	ResendAPIKey       string
	MailpitAddress     string
}

func Load() Config {
	_ = godotenv.Load()
	return Config{
		Environment:        value("APP_ENV", "development"),
		Port:               value("API_PORT", "8080"),
		WebOrigin:          value("WEB_ORIGIN", "http://localhost:5173"),
		DatabaseURL:        value("DATABASE_URL", "postgres://tixigo:tixigo@localhost:5432/tixigo?sslmode=disable"),
		AccessTokenSecret:  value("JWT_ACCESS_SECRET", ""),
		RefreshTokenSecret: value("JWT_REFRESH_SECRET", ""),
		AccessTokenTTL:     duration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:    duration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		EmailFrom:          value("EMAIL_FROM", "Tixigo <noreply@localhost>"),
		ResendAPIKey:       value("RESEND_API_KEY", ""),
		MailpitAddress:     value("MAILPIT_HOST", "localhost") + ":" + value("MAILPIT_PORT", "1025"),
	}
}

func duration(key string, fallback time.Duration) time.Duration {
	parsed, err := time.ParseDuration(os.Getenv(key))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
