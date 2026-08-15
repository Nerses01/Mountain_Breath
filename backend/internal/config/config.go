package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr        string // address the HTTP server listens on, e.g. ":8080"
	Env         string // "dev" or "prod"
	DatabaseURL string // postgres connection string (DSN)
	UploadsDir  string // directory for uploaded product images

	// E8. PublicURL is the ORIGIN THE BROWSER USES (the Vite dev server, or
	// nginx in prod) — it goes into emailed links and OAuth redirect URIs,
	// both of which a visitor's browser must be able to open. The API's own
	// address would be wrong in both places.
	PublicURL string
	// SMTP relay for outgoing mail; empty = messages land in the log
	// instead (mail.LogSink). Dev normally points at Mailpit.
	SMTPAddr     string
	SMTPUsername string
	SMTPPassword string
	MailFrom     string
	// Google OAuth client (decision #5) — created by hand in the Google
	// Cloud console; both empty = the whole flow answers 404 and the login
	// page's button explains itself.
	GoogleClientID     string
	GoogleClientSecret string
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        getEnv("MB_ADDR", ":8080"),
		Env:         getEnv("MB_ENV", "dev"),
		DatabaseURL: os.Getenv("MB_DATABASE_URL"),
		UploadsDir:  getEnv("MB_UPLOADS_DIR", "uploads"),

		PublicURL:          getEnv("MB_PUBLIC_URL", "http://localhost:5173"),
		SMTPAddr:           os.Getenv("MB_SMTP_ADDR"),
		SMTPUsername:       os.Getenv("MB_SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("MB_SMTP_PASSWORD"),
		MailFrom:           getEnv("MB_MAIL_FROM", "Mountain Breath <hive@mountain-breath.local>"),
		GoogleClientID:     os.Getenv("MB_GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("MB_GOOGLE_CLIENT_SECRET"),
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
