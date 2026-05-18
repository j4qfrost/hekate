// Package index materialises raw firehose events into typed Postgres tables
// (venues, slots, events, rsvps). Idempotent on the (did, rkey) primary key.
//
// The Postgres-backed implementation lives in postgres.go.
package index

import "context"

// Writer is the indexer interface. The Postgres-backed implementation is
// PgWriter (see postgres.go); tests can use an in-memory fake.
type Writer interface {
	UpsertVenue(ctx context.Context, v Venue) error
	UpsertSlot(ctx context.Context, s Slot) error
	UpsertEvent(ctx context.Context, e Event) error
	UpsertRSVP(ctx context.Context, r RSVP) error
	UpsertRecurrence(ctx context.Context, rec Recurrence) error
}
