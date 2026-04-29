// Package config loads server configuration from the environment with sane
// defaults. Every knob in this struct corresponds to one HEKATE_* env var
// documented in docs/SELFHOST.md.
package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Listen                string
	RelayURL              string
	DatabaseURL           string
	RecurrenceHorizonDays int
	IndexerBatchSize      int
}

func Load() (*Config, error) {
	cfg := &Config{
		Listen:                envOr("HEKATE_LISTEN", ":8080"),
		RelayURL:              envOr("HEKATE_RELAY_URL", "wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos"),
		DatabaseURL:           envOr("HEKATE_DATABASE_URL", "postgres://hekate:hekate@localhost:5432/hekate?sslmode=disable"),
		RecurrenceHorizonDays: envInt("HEKATE_RECURRENCE_HORIZON_DAYS", 90),
		IndexerBatchSize:      envInt("HEKATE_INDEXER_BATCH_SIZE", 100),
	}
	if cfg.RecurrenceHorizonDays <= 0 {
		return nil, errors.New("HEKATE_RECURRENCE_HORIZON_DAYS must be positive")
	}
	if cfg.IndexerBatchSize <= 0 {
		return nil, errors.New("HEKATE_INDEXER_BATCH_SIZE must be positive")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
