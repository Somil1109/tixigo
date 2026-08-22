package config

import "os"

type Config struct {
	Environment string
	Port        string
	WebOrigin   string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Environment: value("APP_ENV", "development"),
		Port:        value("API_PORT", "8080"),
		WebOrigin:   value("WEB_ORIGIN", "http://localhost:5173"),
		DatabaseURL: value("DATABASE_URL", "postgres://tixigo:tixigo@localhost:5432/tixigo?sslmode=disable"),
	}
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
