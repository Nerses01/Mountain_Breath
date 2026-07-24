package config

import "os"

type Config struct {
	Addr string // address the HTTP server listens on, e.g. ":8080"
	Env  string // "dev" or "prod"
}

func Load() Config {
	return Config{
		Addr: getEnv("MB_ADDR", ":8080"),
		Env:  getEnv("MB_ENV", "dev"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
