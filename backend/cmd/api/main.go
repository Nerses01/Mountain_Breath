package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
	"github.com/Nerses01/Mountain_Breath/backend/internal/config"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// ctx is cancelled the moment the process receives Ctrl+C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: api.NewServer(logger, cfg.Env == "dev").Routes(),
		// Never run an HTTP server without timeouts: a client that sends
		// its request one byte per minute would otherwise hold a goroutine
		// and a connection open forever (slowloris attack).
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ListenAndServe blocks, so it runs in its own goroutine; main keeps
	// going and waits for the shutdown signal below.
	go func() {
		logger.Info("server starting", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done() // block here until Ctrl+C

	// Grace period: stop accepting new connections, wait up to 10s for
	// in-flight requests to finish, then give up and close hard.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("shutting down", "grace_period", "10s")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed, closing hard", "error", err)
		if err := srv.Close(); err != nil {
			logger.Error("hard close failed", "error", err)
		}
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}
