// hekate-verify-w3 compares RisingWave's w3_collision materialised view
// against an independently-computed ground truth from Postgres for the slot-
// collision rule in docs/SPEC.md ("earliest record_created_at wins").
//
// Expected pre-conditions (the Makefile's companion-load target sets these
// up):
//
//   - Postgres has events + events_decoded populated by hekate-fixturegen.
//   - RisingWave has the SQL in companion/sql/ applied; w3_collision exists.
//
// Exits 0 on match, 1 on mismatch or any error.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type winner struct {
	slotURI         string
	did             string
	rkey            string
	recordCreatedAt time.Time
}

func main() {
	pgDSN := flag.String("postgres-dsn", env("POSTGRES_DSN", "postgres://hekate:hekate@localhost:5433/hekate?sslmode=disable"), "Postgres DSN (env: POSTGRES_DSN)")
	rwDSN := flag.String("risingwave-dsn", env("RISINGWAVE_DSN", "postgres://root@localhost:4566/dev?sslmode=disable"), "RisingWave DSN (env: RISINGWAVE_DSN)")
	convergeTimeout := flag.Duration("converge-timeout", 60*time.Second, "Max wait for RisingWave's w3_collision to converge")
	convergeStable := flag.Duration("converge-stable", 3*time.Second, "How long row counts must stay stable to count as converged")
	flag.Parse()

	if err := run(*pgDSN, *rwDSN, *convergeTimeout, *convergeStable); err != nil {
		log.Printf("verify-w3: %v", err)
		os.Exit(1)
	}
	log.Printf("verify-w3: OK")
}

func run(pgDSN, rwDSN string, convergeTimeout, convergeStable time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), convergeTimeout+2*time.Minute)
	defer cancel()

	pg, err := sql.Open("pgx", pgDSN)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer pg.Close()

	rw, err := sql.Open("pgx", rwDSN)
	if err != nil {
		return fmt.Errorf("open risingwave: %w", err)
	}
	defer rw.Close()

	if err := pg.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	if err := rw.PingContext(ctx); err != nil {
		return fmt.Errorf("ping risingwave: %w", err)
	}

	expected, err := postgresGroundTruth(ctx, pg)
	if err != nil {
		return fmt.Errorf("compute ground truth: %w", err)
	}
	log.Printf("verify-w3: ground truth has %d slot winners", len(expected))

	if err := waitForConvergence(ctx, rw, len(expected), convergeTimeout, convergeStable); err != nil {
		return fmt.Errorf("wait for convergence: %w", err)
	}

	actual, err := risingwaveWinners(ctx, rw)
	if err != nil {
		return fmt.Errorf("read w3_collision: %w", err)
	}
	log.Printf("verify-w3: w3_collision has %d slot winners", len(actual))

	return diff(expected, actual)
}

// postgresGroundTruth runs the spec rule directly on events_decoded:
// per slot_uri, the row with the smallest record_created_at wins; ties broken
// by (did, rkey).
func postgresGroundTruth(ctx context.Context, pg *sql.DB) (map[string]winner, error) {
	rows, err := pg.QueryContext(ctx, `
		SELECT DISTINCT ON (slot_uri)
		    slot_uri, did, rkey, record_created_at
		FROM events_decoded
		WHERE collection = 'app.hekate.event'
		  AND slot_uri IS NOT NULL
		ORDER BY slot_uri, record_created_at, did, rkey
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]winner)
	for rows.Next() {
		var w winner
		if err := rows.Scan(&w.slotURI, &w.did, &w.rkey, &w.recordCreatedAt); err != nil {
			return nil, err
		}
		out[w.slotURI] = w
	}
	return out, rows.Err()
}

func risingwaveWinners(ctx context.Context, rw *sql.DB) (map[string]winner, error) {
	rows, err := rw.QueryContext(ctx, `
		SELECT slot_uri, winner_did, winner_rkey, winner_created_at
		FROM w3_collision
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]winner)
	for rows.Next() {
		var w winner
		if err := rows.Scan(&w.slotURI, &w.did, &w.rkey, &w.recordCreatedAt); err != nil {
			return nil, err
		}
		out[w.slotURI] = w
	}
	return out, rows.Err()
}

// waitForConvergence polls w3_collision row count until it equals expectedSlots
// AND has stayed stable for convergeStable. Times out after convergeTimeout.
func waitForConvergence(ctx context.Context, rw *sql.DB, expectedSlots int, timeout, stable time.Duration) error {
	deadline := time.Now().Add(timeout)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()

	var lastCount int
	stableSince := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}

		count, err := rwCount(ctx, rw)
		if err != nil {
			return err
		}

		if count == expectedSlots {
			if stableSince.IsZero() || count != lastCount {
				stableSince = time.Now()
			}
			if time.Since(stableSince) >= stable {
				return nil
			}
		} else {
			stableSince = time.Time{}
		}
		lastCount = count

		if time.Now().After(deadline) {
			return fmt.Errorf("convergence timeout: expected %d, last seen %d", expectedSlots, count)
		}
	}
}

func rwCount(ctx context.Context, rw *sql.DB) (int, error) {
	var n int
	err := rw.QueryRowContext(ctx, `SELECT COUNT(*) FROM w3_collision`).Scan(&n)
	return n, err
}

func diff(expected, actual map[string]winner) error {
	var mismatches []string
	keys := make([]string, 0, len(expected))
	for k := range expected {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, slot := range keys {
		want := expected[slot]
		got, ok := actual[slot]
		if !ok {
			mismatches = append(mismatches, fmt.Sprintf("MISSING in RW: slot=%s want did=%s rkey=%s", slot, want.did, want.rkey))
			continue
		}
		if got.did != want.did || got.rkey != want.rkey || !got.recordCreatedAt.Equal(want.recordCreatedAt) {
			mismatches = append(mismatches, fmt.Sprintf(
				"MISMATCH slot=%s\n  want did=%s rkey=%s created=%s\n  got  did=%s rkey=%s created=%s",
				slot, want.did, want.rkey, want.recordCreatedAt.Format(time.RFC3339Nano),
				got.did, got.rkey, got.recordCreatedAt.Format(time.RFC3339Nano)))
		}
	}
	for slot, w := range actual {
		if _, ok := expected[slot]; !ok {
			mismatches = append(mismatches, fmt.Sprintf("EXTRA in RW: slot=%s did=%s rkey=%s", slot, w.did, w.rkey))
		}
	}

	if len(mismatches) > 0 {
		for _, m := range mismatches {
			log.Println(m)
		}
		return fmt.Errorf("%d slot(s) disagree between Postgres ground truth and RisingWave w3_collision", len(mismatches))
	}
	return nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
