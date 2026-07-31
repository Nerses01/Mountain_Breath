package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
	"github.com/Nerses01/Mountain_Breath/backend/internal/config"
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

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
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

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: api.NewServer(logger, store.New(pool), cfg.Env == "dev",
			store.NewPoolCollector(pool)).Routes(),
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
