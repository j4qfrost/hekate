// Command hekate-server is the venue-first AT Protocol indexer + read API.
//
// It runs the read-API HTTP server plus a long-lived indexer pipeline: an
// ATProto firehose subscription filtered for app.hekate.* records, each
// record decoded against the lexicon and written into Postgres+PostGIS. The
// pipeline reconnects with capped backoff if the relay connection drops.
//
// A daily recurrence expander (materialising app.hekate.slot rows from
// app.hekate.recurrence RRULEs) is planned but not yet wired — it needs a
// recurrence read path first.
//
// All record state lives in user/venue PDSes. The server is read-only — it
// never writes records back to a PDS. See docs/ARCHITECTURE.md.
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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/j4qfrost/hekate/server/internal/api"
	"github.com/j4qfrost/hekate/server/internal/config"
	"github.com/j4qfrost/hekate/server/internal/firehose"
	"github.com/j4qfrost/hekate/server/internal/index"
	"github.com/j4qfrost/hekate/server/internal/indexer"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	// Database pool. Startup hard-fails if Postgres is unreachable — the
	// indexer (and, from M1, the read API) both depend on it.
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database pool init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	err = pool.Ping(pingCtx)
	pingCancel()
	if err != nil {
		slog.Error("database unreachable", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// Indexer pipeline: firehose → decode → Postgres. Reconnects on its own
	// (see runIndexer); runs until ctx is cancelled.
	writer := index.NewPgWriter(pool)
	source := &firehose.IndigoSource{
		RelayURL: cfg.RelayURL,
		Logger:   logger.With("component", "firehose"),
	}
	go runIndexer(ctx, source, writer, cfg.IndexerBatchSize, logger.With("component", "indexer"))

	mux := api.NewRouter()
	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

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

// runIndexer drives the firehose→store pipeline, reconnecting with capped
// exponential backoff whenever a source session ends for a non-shutdown
// reason. A session that stayed healthy for a while resets the backoff so
// only genuine flapping escalates the delay. Returns when ctx is cancelled.
func runIndexer(ctx context.Context, src firehose.Source, w index.Writer, batchSize int, log *slog.Logger) {
	const (
		minBackoff   = 1 * time.Second
		maxBackoff   = 60 * time.Second
		healthyAfter = 2 * time.Minute
	)
	backoff := minBackoff
	for {
		start := time.Now()
		err := indexer.Run(ctx, src, w, batchSize, log)
		if ctx.Err() != nil {
			log.Info("indexer stopped", "reason", "shutdown")
			return
		}
		if time.Since(start) >= healthyAfter {
			backoff = minBackoff
		}
		log.Warn("indexer session ended; reconnecting", "backoff", backoff.String(), "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}
