// Package firehose subscribes to an ATProto relay's subscribeRepos stream,
// filters for app.hekate.* records, and emits decoded record bytes onto a
// sink channel for the indexer to consume.
//
// At v0.1 the indigo IndigoSource is the production source; the Mock in
// mock.go is used by tests and for `hekate-server --demo`.
package firehose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/bluesky-social/indigo/repo"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/j4qfrost/hekate/server/internal/telemetry"
)

var firehoseTracer trace.Tracer = telemetry.Tracer("github.com/j4qfrost/hekate/server/internal/firehose")

// HekateNamespace is the lexicon NSID prefix the indexer cares about. Op
// paths look like "app.hekate.venue/3kabcde…".
const HekateNamespace = "app.hekate."

// Event is one observed PDS commit relevant to hekate. The indexer consumes
// these via Source.Next.
//
// For deletions, Record is nil and CID is empty; the indexer treats this as
// a tombstone for the (DID, Collection, RKey) tuple.
type Event struct {
	DID        string
	Collection string // e.g. "app.hekate.venue"
	RKey       string
	CID        string
	Record     []byte // raw CBOR; indexer decodes against the lexicon. nil for deletes.
	CreatedAt  string // RFC3339 timestamp from the commit's Time field.
}

// Source produces events. The real implementation is IndigoSource;
// tests use Mock.
type Source interface {
	Run(ctx context.Context, sink chan<- Event) error
}

// IndigoSource subscribes to com.atproto.sync.subscribeRepos on RelayURL,
// filters commits for app.hekate.* records, and emits Event values on the
// sink channel. Returns when ctx is cancelled or the stream errors.
//
// The indigo SDK version is pinned in go.mod per ADR 0001.
type IndigoSource struct {
	RelayURL string
	Logger   *slog.Logger
}

// Run dials the relay, registers a callback that decodes hekate records out
// of each commit's CAR, and blocks until ctx is cancelled or the connection
// drops. Callers are responsible for reconnect/backoff.
func (s *IndigoSource) Run(ctx context.Context, sink chan<- Event) error {
	if s.RelayURL == "" {
		return errors.New("firehose: RelayURL is empty")
	}
	log := s.Logger
	if log == nil {
		log = slog.Default().With("component", "firehose")
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, s.RelayURL, nil)
	if err != nil {
		return fmt.Errorf("firehose: dial %s: %w", s.RelayURL, err)
	}
	defer conn.Close()
	log.Info("connected", "relay", s.RelayURL)

	callbacks := &events.RepoStreamCallbacks{
		RepoCommit: func(commit *comatproto.SyncSubscribeRepos_Commit) error {
			return s.handleCommit(ctx, commit, sink, log)
		},
		Error: func(ef *events.ErrorFrame) error {
			log.Warn("error frame", "name", ef.Error, "msg", ef.Message)
			return nil
		},
	}

	sched := sequential.NewScheduler("hekate-firehose", callbacks.EventHandler)
	return events.HandleRepoStream(ctx, conn, sched, log)
}

// handleCommit decodes any app.hekate.* records in the commit and forwards
// them to the sink. Commits that don't touch app.hekate.* return immediately.
func (s *IndigoSource) handleCommit(
	ctx context.Context,
	commit *comatproto.SyncSubscribeRepos_Commit,
	sink chan<- Event,
	log *slog.Logger,
) error {
	if !commitTouchesHekate(commit) {
		return nil
	}
	if len(commit.Blocks) == 0 {
		// tooBig commits or sync-only frames have no payload to decode.
		return nil
	}

	start := time.Now()
	ctx, span := firehoseTracer.Start(ctx, "firehose.handle_commit",
		trace.WithAttributes(
			attribute.String("hekate.did", commit.Repo),
			attribute.Int64("atproto.commit.seq", commit.Seq),
			attribute.Int("atproto.commit.ops", len(commit.Ops)),
		))
	defer func() {
		telemetry.FirehoseHandleDuration.Record(ctx, time.Since(start).Seconds())
		span.End()
	}()

	r, err := repo.ReadRepoFromCar(ctx, bytes.NewReader(commit.Blocks))
	if err != nil {
		// A malformed CAR shouldn't kill the stream; log, count, and skip.
		telemetry.FirehoseDecodeErrors.Add(ctx, 1, metric.WithAttributes(
			attribute.String("reason", "read_car"),
		))
		span.RecordError(err)
		log.Warn("read CAR failed", "did", commit.Repo, "seq", commit.Seq, "err", err)
		return nil
	}

	for _, op := range commit.Ops {
		if !strings.HasPrefix(op.Path, HekateNamespace) {
			continue
		}
		slash := strings.IndexByte(op.Path, '/')
		if slash < 0 {
			continue
		}
		collection := op.Path[:slash]
		rkey := op.Path[slash+1:]

		labels := metric.WithAttributes(
			attribute.String("collection", collection),
			attribute.String("action", op.Action),
		)

		switch op.Action {
		case "create", "update":
			if op.Cid == nil {
				continue
			}
			cc, raw, err := r.GetRecordBytes(ctx, op.Path)
			if err != nil {
				telemetry.FirehoseDecodeErrors.Add(ctx, 1, metric.WithAttributes(
					attribute.String("reason", "get_record"),
					attribute.String("collection", collection),
				))
				log.Warn("get record failed", "did", commit.Repo, "path", op.Path, "err", err)
				continue
			}
			ev := Event{
				DID:        commit.Repo,
				Collection: collection,
				RKey:       rkey,
				CID:        cc.String(),
				Record:     append([]byte(nil), (*raw)...),
				CreatedAt:  commit.Time,
			}
			if err := emit(ctx, sink, ev); err != nil {
				return err
			}
			telemetry.FirehoseEvents.Add(ctx, 1, labels)

		case "delete":
			ev := Event{
				DID:        commit.Repo,
				Collection: collection,
				RKey:       rkey,
				CreatedAt:  commit.Time,
			}
			if err := emit(ctx, sink, ev); err != nil {
				return err
			}
			telemetry.FirehoseEvents.Add(ctx, 1, labels)
		}
	}
	return nil
}

func commitTouchesHekate(c *comatproto.SyncSubscribeRepos_Commit) bool {
	for _, op := range c.Ops {
		if strings.HasPrefix(op.Path, HekateNamespace) {
			return true
		}
	}
	return false
}

func emit(ctx context.Context, sink chan<- Event, ev Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case sink <- ev:
		return nil
	}
}
