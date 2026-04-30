// Command hekate-server is the venue-first AT Protocol indexer + read API.
//
// At v0.1 it does three things in independent goroutines:
//   - Subscribes to an ATProto firehose and filters for app.hekate.* records.
//   - Indexes those records into Postgres+PostGIS.
//   - Daily-ticks a recurrence expander that materialises slot rows from RRULEs.
//
// All state lives in user/venue PDSes. The server is read-only — it never
// writes records back to a PDS. See docs/ARCHITECTURE.md.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/j4qfrost/hekate/server/internal/api"
	"github.com/j4qfrost/hekate/server/internal/config"
	"github.com/j4qfrost/hekate/server/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	slog.Info("starting hekate-server",
		"listen", cfg.Listen,
		"relay", cfg.RelayURL,
		"horizon_days", cfg.RecurrenceHorizonDays,
	)

	telemetryCtx, telemetryCancel := context.WithTimeout(context.Background(), 10*time.Second)
	tp, err := telemetry.Init(telemetryCtx, telemetry.FromEnv())
	telemetryCancel()
	if err != nil {
		slog.Error("telemetry init failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			slog.Warn("telemetry shutdown failed", "err", err)
		}
	}()

	mux := api.NewRouter()

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server crashed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	slog.Info("stopped")
}
