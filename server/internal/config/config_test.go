package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HEKATE_LISTEN", "")
	t.Setenv("HEKATE_RELAY_URL", "")
	t.Setenv("HEKATE_DATABASE_URL", "")
	t.Setenv("HEKATE_RECURRENCE_HORIZON_DAYS", "")
	t.Setenv("HEKATE_INDEXER_BATCH_SIZE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Listen != ":8080" {
		t.Errorf("Listen default mismatch: %q", cfg.Listen)
	}
	if cfg.RecurrenceHorizonDays != 90 {
		t.Errorf("RecurrenceHorizonDays default mismatch: %d", cfg.RecurrenceHorizonDays)
	}
	if cfg.IndexerBatchSize != 100 {
		t.Errorf("IndexerBatchSize default mismatch: %d", cfg.IndexerBatchSize)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("HEKATE_LISTEN", ":9090")
	t.Setenv("HEKATE_RECURRENCE_HORIZON_DAYS", "30")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Listen != ":9090" {
		t.Errorf("override not applied: %q", cfg.Listen)
	}
	if cfg.RecurrenceHorizonDays != 30 {
		t.Errorf("override not applied: %d", cfg.RecurrenceHorizonDays)
	}
}

func TestLoadRejectsBadHorizon(t *testing.T) {
	t.Setenv("HEKATE_RECURRENCE_HORIZON_DAYS", "0")
	if _, err := Load(); err == nil {
		t.Error("expected error for non-positive horizon, got nil")
	}
}
