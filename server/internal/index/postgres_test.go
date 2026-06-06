package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration tests against a real Postgres+PostGIS instance.
//
// Skipped unless HEKATE_TEST_DATABASE_URL points at a fresh, writable
// database that already has the postgis extension available. Suggested local
// setup:
//
//	docker run -d --rm --name hekate-pgtest \
//	  -e POSTGRES_USER=hekate -e POSTGRES_PASSWORD=hekate -e POSTGRES_DB=hekate \
//	  -p 5439:5432 postgis/postgis:16-3.4
//	export HEKATE_TEST_DATABASE_URL=postgres://hekate:hekate@localhost:5439/hekate?sslmode=disable
//
// Each test runs the goose migration inline against a per-test schema so
// they can be run in parallel without cross-talk.
//
// testcontainers-go integration is a follow-up; this shape gates the
// PgWriter from regressing without forcing a dep on every contributor's
// machine.

func mustPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("HEKATE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("HEKATE_TEST_DATABASE_URL not set; skipping postgres integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// freshSchema creates a uniquely-named schema, sets search_path, applies
// migration 0001 against it, and returns the schema name. Cleanup drops it.
func freshSchema(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	schema := "hekate_test_" + strings.ReplaceAll(t.Name(), "/", "_")
	schema = strings.ToLower(schema)

	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`); err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	// Set search_path for this pool so subsequent queries land in the
	// test schema. PostGIS extension stays in public; we still resolve it.
	if _, err := pool.Exec(ctx, `SET search_path TO `+schema+`, public`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	migrationPath := filepath.Join("..", "..", "migrations", "0001_init.sql")
	bs, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	// Strip goose markers; run the Up section directly.
	sql := string(bs)
	if i := strings.Index(sql, "-- +goose Down"); i != -1 {
		sql = sql[:i]
	}
	sql = strings.ReplaceAll(sql, "-- +goose Up", "")
	sql = strings.ReplaceAll(sql, "-- +goose StatementBegin", "")
	sql = strings.ReplaceAll(sql, "-- +goose StatementEnd", "")
	// CREATE EXTENSION is already idempotent in 0001; let it run.

	if _, err := pool.Exec(ctx, sql); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`)
	})
	return schema
}

func TestPgWriter_UpsertVenue_Idempotent(t *testing.T) {
	pool := mustPool(t)
	freshSchema(t, pool)
	w := NewPgWriter(pool)
	ctx := context.Background()

	cap32 := int32(120)
	v := Venue{
		DID: "did:plc:test", RKey: "3kvenue1", Name: "Lighthouse",
		Description: "by the sea", Lat: 35.6812, Lon: 139.7671,
		AddressText: "1-1 Marunouchi", Locality: "Chiyoda",
		Region: "Tokyo", Country: "JP", PostalCode: "100-0005",
		Capacity: &cap32, Amenities: []string{"wifi", "step-free"},
		BookingPolicy: "open", ContactEmail: "host@example.test",
		CreatedAt: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
	}
	if err := w.UpsertVenue(ctx, v); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	// Mutate and re-upsert; should overwrite, not duplicate.
	v.Name = "Lighthouse Annex"
	v.Amenities = []string{"wifi"}
	if err := w.UpsertVenue(ctx, v); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var name string
	var amenities []string
	if err := pool.QueryRow(ctx,
		`SELECT name, amenities FROM venues WHERE did=$1 AND rkey=$2`,
		v.DID, v.RKey).Scan(&name, &amenities); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if name != "Lighthouse Annex" {
		t.Errorf("name = %q, want Lighthouse Annex", name)
	}
	if len(amenities) != 1 || amenities[0] != "wifi" {
		t.Errorf("amenities = %v, want [wifi]", amenities)
	}
}

// TestPgWriter_W3_Collision exercises the "earliest createdAt wins" rule
// from docs/SPEC.md across two events claiming the same slot, with the
// out-of-order arrival pattern that motivates the rule.
func TestPgWriter_W3_Collision(t *testing.T) {
	pool := mustPool(t)
	freshSchema(t, pool)
	w := NewPgWriter(pool)
	ctx := context.Background()

	cap32 := int32(50)
	venue := Venue{
		DID: "did:plc:host", RKey: "3kvenueA", Name: "The Hall",
		Lat: 35.0, Lon: 139.0, Country: "JP",
		BookingPolicy: "open", Capacity: &cap32,
		CreatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := w.UpsertVenue(ctx, venue); err != nil {
		t.Fatalf("venue: %v", err)
	}

	slotStart := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	slot := Slot{
		DID: "did:plc:host", RKey: "3kslotA",
		VenueURI:  "at://did:plc:host/app.hekate.venue/3kvenueA",
		Start:     slotStart, End: slotStart.Add(2 * time.Hour),
		Status:    "open",
		CreatedAt: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
	}
	if err := w.UpsertSlot(ctx, slot); err != nil {
		t.Fatalf("slot: %v", err)
	}

	slotURI := "at://did:plc:host/app.hekate.slot/3kslotA"

	// Event A: arrives FIRST, created LATER. (Loses W3.)
	eventA := Event{
		DID: "did:plc:userA", RKey: "3keventA",
		SlotURI: slotURI, Title: "Late-created A", Visibility: "public",
		CreatedAt: time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC),
	}
	if err := w.UpsertEvent(ctx, eventA); err != nil {
		t.Fatalf("event A: %v", err)
	}

	assertClaim := func(t *testing.T, want string) {
		t.Helper()
		var claim *string
		var status string
		if err := pool.QueryRow(ctx,
			`SELECT claimed_by_uri, status FROM slots WHERE did=$1 AND rkey=$2`,
			slot.DID, slot.RKey).Scan(&claim, &status); err != nil {
			t.Fatalf("read slot: %v", err)
		}
		got := ""
		if claim != nil {
			got = *claim
		}
		if got != want {
			t.Errorf("claimed_by_uri = %q, want %q (status=%s)", got, want, status)
		}
	}

	assertClaim(t, "at://did:plc:userA/app.hekate.event/3keventA")

	// Event B: arrives SECOND, created EARLIER. (Wins W3 — retraction.)
	eventB := Event{
		DID: "did:plc:userB", RKey: "3keventB",
		SlotURI: slotURI, Title: "Early-created B", Visibility: "public",
		CreatedAt: time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC),
	}
	if err := w.UpsertEvent(ctx, eventB); err != nil {
		t.Fatalf("event B: %v", err)
	}
	assertClaim(t, "at://did:plc:userB/app.hekate.event/3keventB")

	// Event C: arrives THIRD, created LATER than B. (Does not win.)
	eventC := Event{
		DID: "did:plc:userC", RKey: "3keventC",
		SlotURI: slotURI, Title: "Even later C", Visibility: "public",
		CreatedAt: time.Date(2026, 5, 6, 8, 0, 0, 0, time.UTC),
	}
	if err := w.UpsertEvent(ctx, eventC); err != nil {
		t.Fatalf("event C: %v", err)
	}
	assertClaim(t, "at://did:plc:userB/app.hekate.event/3keventB")

	// Re-upserting B (idempotent) keeps the claim.
	if err := w.UpsertEvent(ctx, eventB); err != nil {
		t.Fatalf("event B reapply: %v", err)
	}
	assertClaim(t, "at://did:plc:userB/app.hekate.event/3keventB")
}

func TestPgWriter_DeleteByKey(t *testing.T) {
	pool := mustPool(t)
	freshSchema(t, pool)
	w := NewPgWriter(pool)
	ctx := context.Background()

	v := Venue{
		DID: "did:plc:del", RKey: "3kvenueDel", Name: "Doomed",
		Lat: 0, Lon: 0, Country: "JP", BookingPolicy: "open",
		CreatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := w.UpsertVenue(ctx, v); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := w.DeleteByKey(ctx, v.DID, "app.hekate.venue", v.RKey); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM venues WHERE did=$1 AND rkey=$2`,
		v.DID, v.RKey).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("after delete count = %d, want 0", n)
	}
}
