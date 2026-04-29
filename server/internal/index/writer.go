// Package index materialises raw firehose events into typed Postgres tables
// (venues, slots, events, rsvps). Idempotent on the (authoring_did, rkey)
// primary key.
//
// M0: type definitions only. M1 wires Postgres + sqlc-generated queries.
package index

import (
	"context"
	"errors"
)

// Writer is the indexer interface; M1 ships a Postgres-backed implementation.
// The interface deliberately accepts already-decoded records so the Postgres
// implementation can live in a separate file with sqlc-generated types.
type Writer interface {
	UpsertVenue(ctx context.Context, v Venue) error
	UpsertSlot(ctx context.Context, s Slot) error
	UpsertEvent(ctx context.Context, e Event) error
	UpsertRSVP(ctx context.Context, r RSVP) error
	UpsertRecurrence(ctx context.Context, rec Recurrence) error
}

// ErrNotImplemented is returned by the placeholder Postgres writer until M1.
var ErrNotImplemented = errors.New("index: postgres writer lands with M1")

// PostgresWriter is the placeholder Postgres-backed implementation. M1 fills
// it in using the sqlc-generated package under server/queries/.
type PostgresWriter struct{}

func (PostgresWriter) UpsertVenue(_ context.Context, _ Venue) error      { return ErrNotImplemented }
func (PostgresWriter) UpsertSlot(_ context.Context, _ Slot) error        { return ErrNotImplemented }
func (PostgresWriter) UpsertEvent(_ context.Context, _ Event) error      { return ErrNotImplemented }
func (PostgresWriter) UpsertRSVP(_ context.Context, _ RSVP) error        { return ErrNotImplemented }
func (PostgresWriter) UpsertRecurrence(_ context.Context, _ Recurrence) error {
	return ErrNotImplemented
}
