// Package indexer drives the firehose → typed-store pipeline. It consumes
// firehose.Event values, decodes each record's DAG-CBOR body against the
// app.hekate.* lexicon, maps it to the persistence types, and dispatches to
// an index.Writer.
//
// A decode or write failure for a single record is logged and counted, never
// fatal — one poison record must not stall the stream. Source-level failures
// (a dropped relay connection) are returned to the caller, which owns the
// reconnect/backoff policy.
package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bluesky-social/indigo/atproto/atdata"

	"github.com/j4qfrost/hekate/server/internal/firehose"
	"github.com/j4qfrost/hekate/server/internal/index"
	"github.com/j4qfrost/hekate/server/internal/lexicon"
)

// Run consumes one source session. It starts src in a goroutine, drains every
// event src emits into w, and returns whatever src.Run returned once the
// stream ends. It owns the sink channel.
//
// Run returns when the source returns — either ctx cancellation (clean
// shutdown, ctx.Err()) or a stream error the caller should react to. A nil
// logger defaults to slog.Default(); a non-positive batchSize defaults to 1.
func Run(ctx context.Context, src firehose.Source, w index.Writer, batchSize int, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	if batchSize <= 0 {
		batchSize = 1
	}

	sink := make(chan firehose.Event, batchSize)
	srcErr := make(chan error, 1)
	go func() {
		// The source is the sole sender; closing here — after Run returns —
		// is the correct hand-off and lets the range loop below drain any
		// buffered events before we read srcErr.
		srcErr <- src.Run(ctx, sink)
		close(sink)
	}()

	var indexed, failed int64
	for ev := range sink {
		if ctx.Err() != nil {
			// Shutting down — stop indexing. Buffered events are dropped;
			// they re-deliver when the firehose cursor resumes.
			break
		}
		if err := handle(ctx, w, ev); err != nil {
			failed++
			log.Warn("index record failed",
				"did", ev.DID, "collection", ev.Collection, "rkey", ev.RKey, "err", err)
			continue
		}
		indexed++
	}

	err := <-srcErr
	log.Info("indexer session ended", "indexed", indexed, "failed", failed, "err", err)
	return err
}

// handle decodes and dispatches a single firehose event. A nil Record is a
// delete tombstone for the (DID, Collection, RKey) tuple.
func handle(ctx context.Context, w index.Writer, ev firehose.Event) error {
	if ev.Record == nil {
		if err := w.DeleteByKey(ctx, ev.DID, ev.Collection, ev.RKey); err != nil {
			return fmt.Errorf("delete: %w", err)
		}
		return nil
	}

	switch ev.Collection {
	case "app.hekate.venue":
		var rec lexicon.Venue
		if err := decode(ev.Record, &rec); err != nil {
			return err
		}
		v, err := venueToIndex(ev, rec)
		if err != nil {
			return err
		}
		return w.UpsertVenue(ctx, v)

	case "app.hekate.recurrence":
		var rec lexicon.Recurrence
		if err := decode(ev.Record, &rec); err != nil {
			return err
		}
		r, err := recurrenceToIndex(ev, rec)
		if err != nil {
			return err
		}
		return w.UpsertRecurrence(ctx, r)

	case "app.hekate.slot":
		var rec lexicon.Slot
		if err := decode(ev.Record, &rec); err != nil {
			return err
		}
		s, err := slotToIndex(ev, rec)
		if err != nil {
			return err
		}
		return w.UpsertSlot(ctx, s)

	case "app.hekate.event":
		var rec lexicon.Event
		if err := decode(ev.Record, &rec); err != nil {
			return err
		}
		e, err := eventToIndex(ev, rec)
		if err != nil {
			return err
		}
		return w.UpsertEvent(ctx, e)

	case "app.hekate.rsvp":
		var rec lexicon.RSVP
		if err := decode(ev.Record, &rec); err != nil {
			return err
		}
		r, err := rsvpToIndex(ev, rec)
		if err != nil {
			return err
		}
		return w.UpsertRSVP(ctx, r)

	default:
		return fmt.Errorf("unknown collection %q", ev.Collection)
	}
}

// decode converts a record's DAG-CBOR body into a lexicon struct. It routes
// through the atproto data model (CBOR → generic object → JSON) so the
// json-tagged types in package lexicon stay the single decoding contract;
// M1 swaps this for generated codecs (see lexicon package doc / ADR 0001).
func decode(record []byte, into any) error {
	obj, err := atdata.UnmarshalCBOR(record)
	if err != nil {
		return fmt.Errorf("cbor decode: %w", err)
	}
	j, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("re-encode json: %w", err)
	}
	if err := json.Unmarshal(j, into); err != nil {
		return fmt.Errorf("unmarshal lexicon: %w", err)
	}
	return nil
}
