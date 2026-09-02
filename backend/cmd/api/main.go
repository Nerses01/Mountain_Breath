package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
	"github.com/Nerses01/Mountain_Breath/backend/internal/config"
	"github.com/Nerses01/Mountain_Breath/backend/internal/mail"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

func main() {
	// `api healthcheck` probes the running server and exits 0/1. It exists
	// for Docker HEALTHCHECK: the distroless production image has no shell
	// or curl, but it always has this very binary.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// run does the real work; main only translates its error into an exit
	// code. This keeps `defer` working — os.Exit skips deferred calls.
	if err := run(logger); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func healthcheck() int {
	addr := os.Getenv("MB_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost" + addr + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	_ = resp.Body.Close()
	return 0
}

func run(logger *slog.Logger) error {
	// Dev convenience: load backend/.env into the process environment if it
	// exists. In prod there is no .env file — real env vars are set instead.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// ctx is cancelled the moment the process receives Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	// E10's load-test finding, and the single biggest latency fix in the
	// project: Postgres JIT-compiles queries whose ESTIMATED cost crosses
	// jit_above_cost, and variant_effective_prices' NUMERIC/power() math
	// inflates the estimate enough to trigger it — 49 LLVM functions
	// compiled per query, ~250ms of compiler time for a 1ms, 4-row answer,
	// paid on EVERY price read because JIT output is never cached across
	// queries. Under 25 virtual users that compounded to p95 = 3s.
	//
	// JIT exists for analytics — minutes-long scans where compilation
	// amortizes. Every query this shop runs is sub-millisecond OLTP, so it
	// is disabled for the app's connections (not server-wide: someone's
	// future reporting session may well want it). Measured: the catalog
	// p95 fell from 3,090ms to under the 200ms SLO with this one line.
	poolCfg.ConnConfig.RuntimeParams["jit"] = "off"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Fail fast at startup if the DB is unreachable — better than
	// discovering it on the first request.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return err
	}
	logger.Info("database connected")

	if err := os.MkdirAll(cfg.UploadsDir, 0o755); err != nil {
		return fmt.Errorf("creating uploads dir: %w", err)
	}

	// E8: outgoing mail. Configured SMTP (Mailpit in dev, a real relay in
	// prod) or the log sink, so an unconfigured machine still runs and
	// still SHOWS what it would have sent.
	var mailer mail.Mailer = &mail.LogSink{Log: logger}
	if cfg.SMTPAddr != "" {
		mailer = &mail.SMTP{
			Addr: cfg.SMTPAddr, From: cfg.MailFrom,
			Username: cfg.SMTPUsername, Password: cfg.SMTPPassword,
		}
		logger.Info("mail via SMTP", "addr", cfg.SMTPAddr)
	}

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: api.NewServer(logger, store.New(pool), cfg.Env == "dev",
			cfg.UploadsDir, api.Options{
				Mailer:             mailer,
				PublicURL:          cfg.PublicURL,
				GoogleClientID:     cfg.GoogleClientID,
				GoogleClientSecret: cfg.GoogleClientSecret,
			}, store.NewPoolCollector(pool)).Routes(),
		// Never run an HTTP server without timeouts: a client that sends
		// its request one byte per minute would otherwise hold a goroutine
		// and a connection open forever (slowloris attack).
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ListenAndServe blocks, so it runs in its own goroutine; run keeps
	// going and waits for the shutdown signal below.
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err // e.g. port already in use
	case <-ctx.Done(): // Ctrl+C
	}

	// Grace period: stop accepting new connections, wait up to 10s for
	// in-flight requests to finish, then give up and close hard.
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	logger.Info("shutting down", "grace_period", "10s")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return err
	}
	logger.Info("server stopped cleanly")
	return nil
}
