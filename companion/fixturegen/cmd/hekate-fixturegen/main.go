// hekate-fixturegen seeds synthetic firehose events directly into the
// indexer's events + events_decoded tables, bypassing the (still-stubbed)
// IndigoSource. Used by the companion stream-processing stack to produce
// reproducible W1 (occupancy) and W3 (collision) workloads.
//
// Two scenarios drive W3 specifically:
//
//	--scenario=in-order  events arrive sorted by record_created_at (baseline).
//	--scenario=skewed    events deliberately reordered so a later-arriving
//	                     event can have an earlier record_created_at, forcing
//	                     RisingWave to revise prior output.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type config struct {
	dsn             string
	scenario        string
	numVenues       int
	slotsPerVenue   int
	collisionRate   float64
	maxSkew         time.Duration
	seed            int64
	eventsPerSecond int
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.dsn, "dsn", env("FIXTUREGEN_DSN", "postgres://hekate:hekate@localhost:5432/hekate?sslmode=disable"), "Postgres DSN (env: FIXTUREGEN_DSN)")
	flag.StringVar(&cfg.scenario, "scenario", "in-order", "Workload scenario: in-order|skewed")
	flag.IntVar(&cfg.numVenues, "num-venues", 4, "Distinct venues to generate")
	flag.IntVar(&cfg.slotsPerVenue, "slots-per-venue", 8, "Slots per venue")
	flag.Float64Var(&cfg.collisionRate, "collision-rate", 0.5, "Fraction of slots with >1 competing event (0..1)")
	flag.DurationVar(&cfg.maxSkew, "max-skew", 30*time.Second, "Max record_created_at vs. observed_at skew in skewed scenario")
	flag.Int64Var(&cfg.seed, "seed", 1, "RNG seed (deterministic output)")
	flag.IntVar(&cfg.eventsPerSecond, "events-per-second", 0, "Throttle inserts; 0 = unthrottled")
	flag.Parse()

	if cfg.scenario != "in-order" && cfg.scenario != "skewed" {
		fatal("invalid --scenario %q (want in-order|skewed)", cfg.scenario)
	}
	if cfg.collisionRate < 0 || cfg.collisionRate > 1 {
		fatal("--collision-rate must be in [0,1]")
	}

	db, err := sql.Open("pgx", cfg.dsn)
	if err != nil {
		fatal("open db: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		fatal("ping db: %v", err)
	}

	rng := rand.New(rand.NewSource(cfg.seed))
	written, err := run(ctx, db, rng, cfg)
	if err != nil {
		fatal("run: %v", err)
	}
	log.Printf("fixturegen: wrote %d rows (scenario=%s)", written, cfg.scenario)
}

// run plans the workload and writes it. Returns the count of (events,
// events_decoded) row pairs inserted (one row per pair).
func run(ctx context.Context, db *sql.DB, rng *rand.Rand, cfg config) (int, error) {
	plan := buildPlan(rng, cfg)
	// Order rows for insertion. In-order: by record_created_at (== arrival).
	// Skewed: by observed_at — that is the order rows actually land in
	// events_decoded, so a later-arriving row may carry an earlier
	// record_created_at and trigger retraction in RisingWave.
	sort.Slice(plan, func(i, j int) bool {
		if cfg.scenario == "skewed" {
			return plan[i].observedAt.Before(plan[j].observedAt)
		}
		return plan[i].recordCreatedAt.Before(plan[j].recordCreatedAt)
	})

	var throttle <-chan time.Time
	if cfg.eventsPerSecond > 0 {
		t := time.NewTicker(time.Second / time.Duration(cfg.eventsPerSecond))
		defer t.Stop()
		throttle = t.C
	}

	written := 0
	for _, row := range plan {
		if throttle != nil {
			select {
			case <-throttle:
			case <-ctx.Done():
				return written, ctx.Err()
			}
		}
		if err := insertRow(ctx, db, row); err != nil {
			return written, fmt.Errorf("insert row %d: %w", written, err)
		}
		written++
	}
	return written, nil
}

func insertRow(ctx context.Context, db *sql.DB, r plannedRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var seq int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO events (did, collection, rkey, cid, record, record_created_at, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING seq
	`, r.did, r.collection, r.rkey, r.cid, r.record, r.recordCreatedAt, r.observedAt).Scan(&seq)
	if err != nil {
		return fmt.Errorf("insert events: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO events_decoded
		    (seq, did, collection, rkey, record_created_at, observed_at,
		     venue_uri, slot_uri, start_at, end_at, status, title)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, seq, r.did, r.collection, r.rkey, r.recordCreatedAt, r.observedAt,
		nullable(r.venueURI), nullable(r.slotURI),
		nullableTime(r.startAt), nullableTime(r.endAt),
		nullable(r.status), nullable(r.title))
	if err != nil {
		return fmt.Errorf("insert events_decoded: %w", err)
	}

	return tx.Commit()
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

type plannedRow struct {
	did             string
	collection      string
	rkey            string
	cid             string
	record          []byte
	recordCreatedAt time.Time
	observedAt      time.Time

	venueURI string
	slotURI  string
	startAt  time.Time
	endAt    time.Time
	status   string
	title    string
}

// buildPlan generates a deterministic mix of slot and event rows. Slots are
// emitted first (with stable record_created_at), then competing events for a
// fraction of slots (driven by --collision-rate). In skewed mode, event
// observed_at is shifted later than record_created_at by a random amount
// bounded by --max-skew.
func buildPlan(rng *rand.Rand, cfg config) []plannedRow {
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	var rows []plannedRow

	type slotRef struct {
		venueDID string
		slotURI  string
		venueURI string
		startAt  time.Time
		endAt    time.Time
	}
	slots := make([]slotRef, 0, cfg.numVenues*cfg.slotsPerVenue)

	for v := 0; v < cfg.numVenues; v++ {
		venueDID := fmt.Sprintf("did:plc:venue%03d", v)
		venueRkey := fmt.Sprintf("v%03d", v)
		venueURI := atURI(venueDID, "app.hekate.venue", venueRkey)

		venueT := base.Add(time.Duration(v) * time.Minute)
		rows = append(rows, mkRow(rng, venueDID, "app.hekate.venue", venueRkey, venueT, venueT, plannedRow{
			venueURI: venueURI,
		}))

		for s := 0; s < cfg.slotsPerVenue; s++ {
			slotRkey := fmt.Sprintf("s%03d-%03d", v, s)
			slotURI := atURI(venueDID, "app.hekate.slot", slotRkey)
			slotStart := base.Add(time.Duration(s+1) * time.Hour)
			slotEnd := slotStart.Add(time.Hour)
			slotCreated := venueT.Add(time.Duration(s) * time.Second)
			rows = append(rows, mkRow(rng, venueDID, "app.hekate.slot", slotRkey, slotCreated, slotCreated, plannedRow{
				venueURI: venueURI,
				slotURI:  slotURI,
				startAt:  slotStart,
				endAt:    slotEnd,
				status:   "open",
			}))
			slots = append(slots, slotRef{venueDID: venueDID, slotURI: slotURI, venueURI: venueURI, startAt: slotStart, endAt: slotEnd})
		}
	}

	// Events claiming slots. For collisionRate fraction of slots, emit two
	// competing events. Otherwise, one event.
	for i, s := range slots {
		competitors := 1
		if rng.Float64() < cfg.collisionRate {
			competitors = 2
		}
		for c := 0; c < competitors; c++ {
			organizerDID := fmt.Sprintf("did:plc:org%03d-%d", i, c)
			eventRkey := fmt.Sprintf("e%05d-%d", i, c)

			// record_created_at: deterministic per (slot,competitor); spread
			// inside a 60s window so two competitors never tie.
			recordCreated := s.startAt.Add(-time.Duration(60+rng.Intn(60))*time.Second - time.Duration(c*7)*time.Second)

			// observed_at: in-order = record_created; skewed = record_created
			// shifted forward by a uniform draw in [0, max-skew]. With the
			// per-competitor offset above, a later-observed event can carry
			// an earlier record_created_at than its sibling.
			observedAt := recordCreated
			if cfg.scenario == "skewed" {
				observedAt = recordCreated.Add(time.Duration(rng.Int63n(int64(cfg.maxSkew))))
			}

			rows = append(rows, mkRow(rng, organizerDID, "app.hekate.event", eventRkey, recordCreated, observedAt, plannedRow{
				slotURI: s.slotURI,
				title:   fmt.Sprintf("Event %d-%d", i, c),
			}))
		}
	}

	return rows
}

func mkRow(rng *rand.Rand, did, collection, rkey string, recordCreated, observedAt time.Time, partial plannedRow) plannedRow {
	r := partial
	r.did = did
	r.collection = collection
	r.rkey = rkey
	r.recordCreatedAt = recordCreated
	r.observedAt = observedAt
	// Synthetic record bytes — the decoded projection is what RisingWave
	// reads, so the raw blob's content is incidental. Keep it stable per
	// (did, collection, rkey) so re-runs of the same seed produce the same
	// CID and don't trip the events_record_idx unique constraint.
	payload := []byte(fmt.Sprintf("%s|%s|%s|%d", did, collection, rkey, rng.Int63()))
	r.record = payload
	sum := sha256.Sum256(payload)
	r.cid = "fakecid:" + hex.EncodeToString(sum[:8])
	return r
}

func atURI(did, collection, rkey string) string {
	return fmt.Sprintf("at://%s/%s/%s", did, collection, rkey)
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "fixturegen: "+format+"\n", args...)
	os.Exit(1)
}
