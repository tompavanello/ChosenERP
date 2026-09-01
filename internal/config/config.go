package config

import (
	"os"
	"strconv"
	"time"
)

// Config agrega as variáveis de ambiente da API.
type Config struct {
	HTTPPort         string
	DatabaseURL      string
	MigrateURL       string
	JWTSecret        string
	JWTAccessExpiry  time.Duration
	JWTRefreshExpiry time.Duration
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func Load() Config {
	return Config{
		HTTPPort:         getenv("API_HTTP_PORT", "38080"),
		DatabaseURL:      getenv("DATABASE_URL", "postgres://chosenerp_app:chosenapp@127.0.0.1:35432/chosenerp?sslmode=disable"),
		MigrateURL:       getenv("MIGRATE_DATABASE_URL", "postgres://postgres:sinc@127.0.0.1:35432/chosenerp?sslmode=disable"),
		JWTSecret:        getenv("JWT_SECRET", "dev-insecure-secret-change-me"),
		JWTAccessExpiry:  time.Duration(getenvInt("JWT_EXPIRATION_MINUTES", 15)) * time.Minute,
		JWTRefreshExpiry: time.Duration(getenvInt("REFRESH_EXPIRATION_HOURS", 168)) * time.Hour,
	}
}
