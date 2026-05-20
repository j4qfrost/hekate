package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atdata"

	"github.com/j4qfrost/hekate/server/internal/firehose"
	"github.com/j4qfrost/hekate/server/internal/index"
	"github.com/j4qfrost/hekate/server/internal/lexicon"
)

// fakeWriter is an in-memory index.Writer for exercising the indexer loop
// without a Postgres instance.
type fakeWriter struct {
	venues  []index.Venue
	slots   []index.Slot
	events  []index.Event
	rsvps   []index.RSVP
	recurs  []index.Recurrence
	deletes []string // "collection did/rkey"
}

func (f *fakeWriter) UpsertVenue(_ context.Context, v index.Venue) error {
	f.venues = append(f.venues, v)
	return nil
}

func (f *fakeWriter) UpsertSlot(_ context.Context, s index.Slot) error {
	f.slots = append(f.slots, s)
	return nil
}

func (f *fakeWriter) UpsertEvent(_ context.Context, e index.Event) error {
	f.events = append(f.events, e)
	return nil
}

func (f *fakeWriter) UpsertRSVP(_ context.Context, r index.RSVP) error {
	f.rsvps = append(f.rsvps, r)
	return nil
}

func (f *fakeWriter) UpsertRecurrence(_ context.Context, rec index.Recurrence) error {
	f.recurs = append(f.recurs, rec)
	return nil
}

func (f *fakeWriter) DeleteByKey(_ context.Context, did, collection, rkey string) error {
	f.deletes = append(f.deletes, collection+" "+did+"/"+rkey)
	return nil
}

var _ index.Writer = (*fakeWriter)(nil)

func mustCBOR(t *testing.T, obj map[string]any) []byte {
	t.Helper()
	b, err := atdata.MarshalCBOR(obj)
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	return b
}

// TestRun_DecodesAndDispatches drives the full pipeline — CBOR decode, lexicon
// mapping, and Writer dispatch — through a Mock source, with no Postgres. It
// also pins the CBOR round-trip the indexer relies on (atdata MarshalCBOR →
// UnmarshalCBOR → JSON re-encode into the lexicon types).
func TestRun_DecodesAndDispatches(t *testing.T) {
	venueRec := mustCBOR(t, map[string]any{
		"$type":         "app.hekate.venue",
		"name":          "Lighthouse",
		"geo":           map[string]any{"lat": "35.6812", "lon": "139.7671"},
		"bookingPolicy": "open",
		"amenities":     []any{"wifi", "step-free"},
		"createdAt":     "2026-05-01T09:00:00Z",
	})
	recurrenceRec := mustCBOR(t, map[string]any{
		"$type":               "app.hekate.recurrence",
		"venue":               "at://did:plc:host/app.hekate.venue/v1",
		"rrule":               "FREQ=WEEKLY;BYDAY=MO",
		"slotDurationMinutes": int64(60),
		"validFrom":           "2026-05-01T00:00:00Z",
		"createdAt":           "2026-05-01T08:00:00Z",
	})
	eventRec := mustCBOR(t, map[string]any{
		"$type":     "app.hekate.event",
		"slot":      "at://did:plc:host/app.hekate.slot/3kslotA",
		"title":     "Launch night",
		"createdAt": "2026-05-03T10:00:00Z",
		// visibility omitted -> mapper defaults it to "public"
	})

	mock := &firehose.Mock{Events: []firehose.Event{
		{DID: "did:plc:v", Collection: "app.hekate.venue", RKey: "v1", Record: venueRec},
		{DID: "did:plc:r", Collection: "app.hekate.recurrence", RKey: "r1", Record: recurrenceRec},
		{DID: "did:plc:e", Collection: "app.hekate.event", RKey: "e1", Record: eventRec},
		{DID: "did:plc:v", Collection: "app.hekate.venue", RKey: "v1"}, // delete tombstone
	}}

	w := &fakeWriter{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := Run(ctx, mock, w, 8, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(w.venues) != 1 {
		t.Fatalf("venues = %d, want 1", len(w.venues))
	}
	v := w.venues[0]
	if v.DID != "did:plc:v" || v.RKey != "v1" || v.Name != "Lighthouse" {
		t.Errorf("venue identity/name wrong: %+v", v)
	}
	if v.Lat != 35.6812 || v.Lon != 139.7671 {
		t.Errorf("venue coordinates not parsed from strings: lat=%v lon=%v", v.Lat, v.Lon)
	}

	if len(w.recurs) != 1 {
		t.Fatalf("recurrences = %d, want 1", len(w.recurs))
	}
	r := w.recurs[0]
	if r.DID != "did:plc:r" || r.RRule != "FREQ=WEEKLY;BYDAY=MO" || r.SlotDurationMinutes != 60 {
		t.Errorf("recurrence mapped wrong: %+v", r)
	}
	if !r.ValidFrom.Equal(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("recurrence validFrom = %v", r.ValidFrom)
	}

	if len(w.events) != 1 {
		t.Fatalf("events = %d, want 1", len(w.events))
	}
	if e := w.events[0]; e.DID != "did:plc:e" || e.SlotURI != "at://did:plc:host/app.hekate.slot/3kslotA" {
		t.Errorf("event identity/slot wrong: %+v", e)
	}
	if w.events[0].Visibility != "public" {
		t.Errorf("event visibility = %q, want defaulted to public", w.events[0].Visibility)
	}

	if len(w.deletes) != 1 || w.deletes[0] != "app.hekate.venue did:plc:v/v1" {
		t.Errorf("deletes = %v, want one venue tombstone", w.deletes)
	}
}

// TestRun_PoisonRecordDoesNotStall verifies that a record which fails to
// decode is skipped, not fatal — the events after it still index.
func TestRun_PoisonRecordDoesNotStall(t *testing.T) {
	goodRec := mustCBOR(t, map[string]any{
		"$type":               "app.hekate.recurrence",
		"venue":               "at://did:plc:host/app.hekate.venue/v1",
		"rrule":               "FREQ=WEEKLY",
		"slotDurationMinutes": int64(30),
		"validFrom":           "2026-05-01T00:00:00Z",
		"createdAt":           "2026-05-01T08:00:00Z",
	})

	mock := &firehose.Mock{Events: []firehose.Event{
		{DID: "did:plc:bad", Collection: "app.hekate.recurrence", RKey: "bad", Record: []byte("not cbor")},
		{DID: "did:plc:good", Collection: "app.hekate.recurrence", RKey: "good", Record: goodRec},
	}}

	w := &fakeWriter{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := Run(ctx, mock, w, 4, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(w.recurs) != 1 || w.recurs[0].RKey != "good" {
		t.Errorf("poison record stalled the stream: recurrences = %+v", w.recurs)
	}
}

// TestVenueToIndex covers the venue lexicon→persistence mapping directly,
// including the decimal-string coordinate parse and the address/contact
// flattening.
func TestVenueToIndex(t *testing.T) {
	capacity := 120
	altitude := "12.5"
	rec := lexicon.Venue{
		Type:        "app.hekate.venue",
		Name:        "Lighthouse",
		Description: "by the sea",
		Geo:         lexicon.GeoPoint{Lat: "35.6812", Lon: "139.7671", AltitudeMeters: &altitude},
		Address: &lexicon.PostalAddress{
			Text: "1-1 Marunouchi", Locality: "Chiyoda",
			Region: "Tokyo", Country: "JP", PostalCode: "100-0005",
		},
		Capacity:      &capacity,
		Amenities:     []string{"wifi", "step-free"},
		BookingPolicy: "open",
		Contact:       &lexicon.ContactInfo{Email: "host@example.test", URL: "https://example.test"},
		CreatedAt:     "2026-05-01T09:00:00Z",
	}

	v, err := venueToIndex(firehose.Event{DID: "did:plc:v", RKey: "v1"}, rec)
	if err != nil {
		t.Fatalf("venueToIndex: %v", err)
	}
	if v.DID != "did:plc:v" || v.RKey != "v1" {
		t.Errorf("identity wrong: %+v", v)
	}
	if v.Lat != 35.6812 || v.Lon != 139.7671 {
		t.Errorf("coordinates not parsed from strings: lat=%v lon=%v", v.Lat, v.Lon)
	}
	if v.AltitudeMeters == nil || *v.AltitudeMeters != 12.5 {
		t.Errorf("altitude not parsed: %v", v.AltitudeMeters)
	}
	if v.Locality != "Chiyoda" || v.Region != "Tokyo" || v.Country != "JP" || v.PostalCode != "100-0005" {
		t.Errorf("address not flattened: %+v", v)
	}
	if v.ContactEmail != "host@example.test" || v.ContactURL != "https://example.test" {
		t.Errorf("contact not flattened: email=%q url=%q", v.ContactEmail, v.ContactURL)
	}
	if v.Capacity == nil || *v.Capacity != 120 {
		t.Errorf("capacity not narrowed to int32: %v", v.Capacity)
	}
	if !v.CreatedAt.Equal(time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)) {
		t.Errorf("createdAt = %v", v.CreatedAt)
	}

	// A venue with no address / contact must leave those fields zero-valued,
	// not panic on the nil pointers.
	bare, err := venueToIndex(firehose.Event{DID: "did:plc:v", RKey: "v2"}, lexicon.Venue{
		Name: "Bare", Geo: lexicon.GeoPoint{Lat: "1", Lon: "2"},
		BookingPolicy: "open", CreatedAt: "2026-05-01T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("venueToIndex (bare): %v", err)
	}
	if bare.Locality != "" || bare.ContactEmail != "" {
		t.Errorf("bare venue should have empty address/contact: %+v", bare)
	}
}

// TestVenueToIndex_BadCoordinates verifies a venue with an unparseable or
// out-of-range coordinate is rejected (becomes a per-record decode error the
// indexer logs and skips), rather than silently storing garbage.
func TestVenueToIndex_BadCoordinates(t *testing.T) {
	cases := map[string]lexicon.GeoPoint{
		"non-numeric lat": {Lat: "north", Lon: "139.7"},
		"lat out of range": {Lat: "95.0", Lon: "139.7"},
		"lon out of range": {Lat: "35.0", Lon: "200.0"},
	}
	for name, geo := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := venueToIndex(firehose.Event{DID: "did:plc:v", RKey: "v1"}, lexicon.Venue{
				Name: "Bad", Geo: geo, BookingPolicy: "open",
				CreatedAt: "2026-05-01T09:00:00Z",
			})
			if err == nil {
				t.Errorf("expected an error for %s, got nil", name)
			}
		})
	}
}
