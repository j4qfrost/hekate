// Package firehose subscribes to an ATProto relay's subscribeRepos stream,
// filters for app.hekate.* records, and writes raw events into the indexer's
// events table.
//
// M1 will wire this up against github.com/bluesky-social/indigo/events. For
// now it defines the interface and a Mock implementation usable by tests.
package firehose

import (
	"context"
	"errors"
)

// Event is one observed PDS commit relevant to hekate. The indexer consumes
// these via Source.Next.
type Event struct {
	DID        string
	Collection string // e.g. "app.hekate.venue"
	RKey       string
	CID        string
	Record     []byte // raw CBOR or JSON; indexer decodes against the lexicon
	CreatedAt  string // RFC3339 from the record itself, not the firehose
}

// Source produces events. The real implementation is a websocket loop;
// tests use the Mock below.
type Source interface {
	Run(ctx context.Context, sink chan<- Event) error
}

// ErrNotImplemented is returned by the placeholder real source until M1.
var ErrNotImplemented = errors.New("firehose: indigo subscriber lands with M1")

// IndigoSource is a placeholder for the real subscribeRepos consumer.
// It will be implemented against indigo/events in M1; per ADR 0001 the
// indigo version is pinned in go.mod.
type IndigoSource struct {
	RelayURL string
}

func (s *IndigoSource) Run(_ context.Context, _ chan<- Event) error {
	return ErrNotImplemented
}
