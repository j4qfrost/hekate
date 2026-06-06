// Package indexer's external PDS round-trip integration test. This file is
// the literal M1 kill-criterion deliverable: "reads at least one external
// PDS's records end-to-end" (see ~/.claude/plans/create-project-that-helps-streamed-platypus.md
// and ../../../CLAUDE.md). If this test does not exist and pass by
// 2026-10-29, the project scopes down to lexicons-only.
//
// Why a separate file: gating *all* indexer tests behind a build tag would
// hide the unit-level pipeline coverage in indexer_test.go. Go's
// per-file build-tag mechanism lets the integration test live in the same
// package without entangling the fast tests.
//
// PDS choice: bsky.social.
//   - PubNet (~/infra/pubnet/) is a Dart pub registry, not an AT Protocol
//     PDS — it cannot host app.hekate.* records.
//   - bsky.social is the canonical reference PDS, broadcasts to the
//     bsky.network relay this indexer subscribes to by default, and
//     accepts arbitrary lexicon collections (validation = warn-only for
//     unknown lexicons).
//   - app.hekate.venue is the only collection exercised. ADR 0002 leaves
//     app.hekate.event / app.hekate.rsvp subject to change pending Smoke
//     Signal coordination, and venue is uniquely hekate's contribution
//     regardless of how that ADR closes. Per CLAUDE.md ("Don't ship
//     app.hekate.event or app.hekate.rsvp from a non-test PDS until
//     ADR 0002 closes"), exercising venue here satisfies the kill
//     criterion without preempting the open ADR.
//
// Lexicon coordination follow-up (deferred): once ADR 0002 closes,
// extend this test to also publish app.hekate.event and assert the W3
// collision rule end-to-end against a real firehose. Tracked in the PR.
//
//go:build integration

package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/xrpc"

	"github.com/j4qfrost/hekate/server/internal/firehose"
	"github.com/j4qfrost/hekate/server/internal/index"
)

const (
	envPDSURL      = "HEKATE_TEST_PDS_URL"
	envHandle      = "HEKATE_TEST_HANDLE"
	envAppPassword = "HEKATE_TEST_APP_PASSWORD"
	envRelayURL    = "HEKATE_TEST_RELAY_URL"
	envTimeout     = "HEKATE_TEST_TIMEOUT"

	defaultPDSURL   = "https://bsky.social"
	defaultRelayURL = "wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos"
	defaultTimeout  = 180 * time.Second
)

// recordingWriter is an index.Writer that captures every upsert and lets the
// test rendezvous on the first venue whose (DID, RKey) matches the record we
// just published.
type recordingWriter struct {
	mu        sync.Mutex
	venues    []index.Venue
	venueGate chan index.Venue // unbuffered; sent on a match, never blocks the indexer because consumers drain it.
	match     struct {
		did, rkey string
	}
}

func newRecordingWriter(did, rkey string) *recordingWriter {
	w := &recordingWriter{venueGate: make(chan index.Venue, 1)}
	w.match.did, w.match.rkey = did, rkey
	return w
}

func (w *recordingWriter) UpsertVenue(_ context.Context, v index.Venue) error {
	w.mu.Lock()
	w.venues = append(w.venues, v)
	w.mu.Unlock()
	if v.DID == w.match.did && v.RKey == w.match.rkey {
		// Best-effort, non-blocking: indexer must not stall on test sync.
		select {
		case w.venueGate <- v:
		default:
		}
	}
	return nil
}

func (w *recordingWriter) UpsertSlot(_ context.Context, _ index.Slot) error             { return nil }
func (w *recordingWriter) UpsertEvent(_ context.Context, _ index.Event) error           { return nil }
func (w *recordingWriter) UpsertRSVP(_ context.Context, _ index.RSVP) error             { return nil }
func (w *recordingWriter) UpsertRecurrence(_ context.Context, _ index.Recurrence) error { return nil }
func (w *recordingWriter) DeleteByKey(_ context.Context, _, _, _ string) error          { return nil }

var _ index.Writer = (*recordingWriter)(nil)

// requireEnv reads the three mandatory test env vars and skips the test if
// any are unset — the standard "live-test opt-in" pattern.
func requireEnv(t *testing.T) (pdsURL, relayURL, handle, password string, timeout time.Duration) {
	t.Helper()
	handle = os.Getenv(envHandle)
	password = os.Getenv(envAppPassword)
	if handle == "" || password == "" {
		t.Skipf("set %s and %s (with optional %s, %s) to run the external PDS round-trip test",
			envHandle, envAppPassword, envPDSURL, envRelayURL)
	}
	pdsURL = os.Getenv(envPDSURL)
	if pdsURL == "" {
		pdsURL = defaultPDSURL
	}
	relayURL = os.Getenv(envRelayURL)
	if relayURL == "" {
		relayURL = defaultRelayURL
	}
	timeout = defaultTimeout
	if v := os.Getenv(envTimeout); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			t.Fatalf("%s: %v", envTimeout, err)
		}
		timeout = d
	}
	return
}

// newXRPCSession authenticates against the PDS with an app password and
// returns a client whose Auth/Host are populated. The session is short-lived
// and one-shot — no refresh logic, app-password sessions are cheap to mint.
func newXRPCSession(ctx context.Context, pdsURL, handle, password string) (*xrpc.Client, error) {
	c := &xrpc.Client{
		Client: &http.Client{Timeout: 30 * time.Second},
		Host:   pdsURL,
	}
	out, err := comatproto.ServerCreateSession(ctx, c, &comatproto.ServerCreateSession_Input{
		Identifier: handle,
		Password:   password,
	})
	if err != nil {
		return nil, fmt.Errorf("createSession %s: %w", pdsURL, err)
	}
	c.Auth = &xrpc.AuthInfo{
		AccessJwt:  out.AccessJwt,
		RefreshJwt: out.RefreshJwt,
		Handle:     out.Handle,
		Did:        out.Did,
	}
	return c, nil
}

// createVenueRecord publishes an app.hekate.venue record by speaking XRPC
// directly with a map payload. We bypass the indigo lexicon registry
// (which only knows com.atproto / app.bsky lexicons) by using the raw
// xrpc.Client.Do path with a JSON body.
//
// Returns the AT-URI and the rkey assigned by the PDS.
func createVenueRecord(ctx context.Context, c *xrpc.Client, did string) (uri, rkey string, err error) {
	// Minimal valid venue per lexicons/app/hekate/venue.json: name, geo,
	// bookingPolicy, createdAt are required; geo lat/lon are decimal-degree
	// strings (the AT Protocol data model has no float type — see
	// lexicon.GeoPoint).
	createdAt := time.Now().UTC().Format(time.RFC3339)
	body := map[string]any{
		"repo":       did,
		"collection": "app.hekate.venue",
		"record": map[string]any{
			"$type":         "app.hekate.venue",
			"name":          "hekate integration test venue",
			"description":   "Auto-published by hekate's M1 external-PDS round-trip integration test. Deleted on test teardown.",
			"geo":           map[string]any{"lat": "0.0", "lon": "0.0"},
			"bookingPolicy": "open",
			"amenities":     []string{"wifi"},
			"createdAt":     createdAt,
		},
	}
	var out struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}
	if err := c.Do(ctx, xrpc.Procedure, "application/json", "com.atproto.repo.createRecord", nil, body, &out); err != nil {
		return "", "", fmt.Errorf("createRecord: %w", err)
	}
	// AT-URI shape: at://<did>/<collection>/<rkey> — split on the final slash.
	rk, err := rkeyFromURI(out.URI)
	if err != nil {
		return out.URI, "", err
	}
	return out.URI, rk, nil
}

func rkeyFromURI(uri string) (string, error) {
	// Cheap parse: last path segment. Avoids pulling in syntax.ATURI just for this.
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == '/' {
			return uri[i+1:], nil
		}
	}
	return "", fmt.Errorf("malformed AT-URI %q", uri)
}

func deleteVenueRecord(ctx context.Context, c *xrpc.Client, did, rkey string) error {
	_, err := comatproto.RepoDeleteRecord(ctx, c, &comatproto.RepoDeleteRecord_Input{
		Repo:       did,
		Collection: "app.hekate.venue",
		Rkey:       rkey,
	})
	return err
}

// TestExternalPDSRoundTrip is the M1 kill-criterion test.
//
// Flow:
//  1. Authenticate against the configured PDS with an app password.
//  2. Start the firehose IndigoSource against the configured relay and a
//     recording writer.
//  3. Publish an app.hekate.venue record to the PDS.
//  4. Wait (with timeout) for the indexer to observe and decode the
//     record off the firehose.
//  5. Assert the indexed venue matches what we published, by (DID, RKey)
//     identity and by core fields (name, geo).
//  6. Clean up the record from the PDS.
//
// Opt-in: requires HEKATE_TEST_HANDLE + HEKATE_TEST_APP_PASSWORD. Optional
// HEKATE_TEST_PDS_URL (default https://bsky.social), HEKATE_TEST_RELAY_URL
// (default wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos),
// HEKATE_TEST_TIMEOUT (default 180s — firehose lag + relay propagation).
//
// The default 180s timeout accounts for the bsky.network relay's typical
// commit-to-broadcast latency (single-digit seconds in steady state, but
// catch-up reads after the cursor lands can stretch the first event under
// load). Override via HEKATE_TEST_TIMEOUT when running locally against a
// quieter PDS.
func TestExternalPDSRoundTrip(t *testing.T) {
	pdsURL, relayURL, handle, password, timeout := requireEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	t.Logf("M1 round-trip: pds=%s relay=%s handle=%s timeout=%s", pdsURL, relayURL, handle, timeout)

	// 1. Authenticate.
	sess, err := newXRPCSession(ctx, pdsURL, handle, password)
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	did := sess.Auth.Did
	t.Logf("authenticated as did=%s", did)

	// 2. Start firehose + indexer with a recording writer keyed on the
	//    DID we just authenticated as. The rkey is unknown until step 3,
	//    so we set it after the publish succeeds.
	w := &recordingWriter{venueGate: make(chan index.Venue, 1)}
	w.match.did = did
	src := &firehose.IndigoSource{RelayURL: relayURL}
	indexerErr := make(chan error, 1)
	indexerCtx, stopIndexer := context.WithCancel(ctx)
	defer stopIndexer()
	go func() { indexerErr <- Run(indexerCtx, src, w, 16, nil) }()

	// Give the firehose a beat to dial the relay so the venue we publish
	// next isn't dropped before the subscription is live. The actual
	// dial-and-subscribe completes well under 2s against bsky.network.
	time.Sleep(2 * time.Second)

	// 3. Publish.
	uri, rkey, err := createVenueRecord(ctx, sess, did)
	if err != nil {
		t.Fatalf("createRecord: %v", err)
	}
	w.mu.Lock()
	w.match.rkey = rkey
	w.mu.Unlock()
	t.Logf("published venue uri=%s rkey=%s", uri, rkey)

	// Always try to clean up, even on assertion failure.
	defer func() {
		// Use a fresh context — the test's may already be cancelled.
		cleanup, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelCleanup()
		if err := deleteVenueRecord(cleanup, sess, did, rkey); err != nil {
			t.Logf("cleanup deleteRecord (best-effort): %v", err)
		} else {
			t.Logf("cleanup deleteRecord OK")
		}
	}()

	// 4. Wait for the indexer to observe it.
	select {
	case v := <-w.venueGate:
		// 5. Assert content.
		if v.DID != did {
			t.Errorf("indexed venue DID = %q, want %q", v.DID, did)
		}
		if v.RKey != rkey {
			t.Errorf("indexed venue RKey = %q, want %q", v.RKey, rkey)
		}
		if v.Name != "hekate integration test venue" {
			t.Errorf("indexed venue Name = %q, want %q", v.Name, "hekate integration test venue")
		}
		if v.Lat != 0.0 || v.Lon != 0.0 {
			t.Errorf("indexed venue coords = (%v, %v), want (0, 0)", v.Lat, v.Lon)
		}
		if v.BookingPolicy != "open" {
			t.Errorf("indexed venue BookingPolicy = %q, want %q", v.BookingPolicy, "open")
		}
		t.Logf("M1 KILL-CRITERION SATISFIED: external PDS record round-tripped through firehose+indexer in <%s", timeout)
	case <-ctx.Done():
		// Dump what the writer saw so failure modes are diagnosable.
		w.mu.Lock()
		seen := make([]string, 0, len(w.venues))
		for _, v := range w.venues {
			seen = append(seen, v.DID+"/"+v.RKey+"="+v.Name)
		}
		w.mu.Unlock()
		seenJSON, _ := json.Marshal(seen)
		t.Fatalf("M1 kill-criterion test timed out after %s waiting for venue %s/%s; firehose saw %d venues total: %s",
			timeout, did, rkey, len(seen), seenJSON)
	}

	// 6. Stop the indexer cleanly.
	stopIndexer()
	select {
	case <-indexerErr:
	case <-time.After(5 * time.Second):
		t.Log("indexer did not return within 5s of cancellation (non-fatal)")
	}
}
