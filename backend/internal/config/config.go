package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr        string // address the HTTP server listens on, e.g. ":8080"
	Env         string // "dev" or "prod"
	DatabaseURL string // postgres connection string (DSN)
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        getEnv("MB_ADDR", ":8080"),
		Env:         getEnv("MB_ENV", "dev"),
		DatabaseURL: os.Getenv("MB_DATABASE_URL"),
	}
	// No default for the DSN: it contains credentials, which never belong
	// in source code. Fail loudly instead (same idea as compose's ${VAR:?}).
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("MB_DATABASE_URL is not set (dev: put it in backend/.env)")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
