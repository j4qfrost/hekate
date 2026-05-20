// Postgres-backed index.Writer. Idempotent upserts keyed on (did, rkey).
//
// Geography literals: PostGIS GEOGRAPHY(POINT,4326) values are written via
// ST_SetSRID(ST_MakePoint(lon, lat), 4326)::geography. lat/lon order matters
// for ST_MakePoint (it is lon, lat — the only PostGIS function with that
// signature; everything else takes lat first).
//
// State-transition enforcement (per docs/SPEC.md) lives at the SQL layer:
// slot status CHECK constraint rejects unknown values, and the W3 collision
// rule (earliest record.createdAt wins on slot claim) is handled by the
// claim-event upsert that conditionally updates slots.claimed_by_uri only
// when the incoming event's createdAt beats the current claim.
package index

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgWriter holds a pgxpool.Pool. Construct with NewPgWriter and pass to the
// indexer goroutine. Cheap to copy by value (the pool inside is a pointer).
type PgWriter struct {
	pool *pgxpool.Pool
}

func NewPgWriter(pool *pgxpool.Pool) *PgWriter {
	return &PgWriter{pool: pool}
}

// Ensure PgWriter satisfies the Writer interface at compile time.
var _ Writer = (*PgWriter)(nil)

func (w *PgWriter) UpsertVenue(ctx context.Context, v Venue) error {
	if v.Amenities == nil {
		v.Amenities = []string{}
	}
	const q = `
		INSERT INTO venues (
		    did, rkey, name, description,
		    geo, altitude_meters,
		    address_text, locality, region, country, postal_code,
		    capacity, amenities, booking_policy,
		    contact_email, contact_url, created_at
		) VALUES (
		    $1, $2, $3, $4,
		    ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography, $7,
		    $8, $9, $10, $11, $12,
		    $13, $14, $15,
		    $16, $17, $18
		)
		ON CONFLICT (did, rkey) DO UPDATE SET
		    name             = EXCLUDED.name,
		    description      = EXCLUDED.description,
		    geo              = EXCLUDED.geo,
		    altitude_meters  = EXCLUDED.altitude_meters,
		    address_text     = EXCLUDED.address_text,
		    locality         = EXCLUDED.locality,
		    region           = EXCLUDED.region,
		    country          = EXCLUDED.country,
		    postal_code      = EXCLUDED.postal_code,
		    capacity         = EXCLUDED.capacity,
		    amenities        = EXCLUDED.amenities,
		    booking_policy   = EXCLUDED.booking_policy,
		    contact_email    = EXCLUDED.contact_email,
		    contact_url      = EXCLUDED.contact_url,
		    created_at       = EXCLUDED.created_at
	`
	_, err := w.pool.Exec(ctx, q,
		v.DID, v.RKey, v.Name, v.Description,
		v.Lon, v.Lat, v.AltitudeMeters,
		v.AddressText, v.Locality, v.Region, v.Country, v.PostalCode,
		v.Capacity, v.Amenities, v.BookingPolicy,
		v.ContactEmail, v.ContactURL, v.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert venue %s/%s: %w", v.DID, v.RKey, err)
	}
	return nil
}

func (w *PgWriter) UpsertRecurrence(ctx context.Context, r Recurrence) error {
	const q = `
		INSERT INTO recurrences (
		    did, rkey, venue_uri, rrule,
		    slot_duration_minutes, title,
		    valid_from, valid_until, created_at
		) VALUES (
		    $1, $2, $3, $4,
		    $5, $6,
		    $7, $8, $9
		)
		ON CONFLICT (did, rkey) DO UPDATE SET
		    venue_uri              = EXCLUDED.venue_uri,
		    rrule                  = EXCLUDED.rrule,
		    slot_duration_minutes  = EXCLUDED.slot_duration_minutes,
		    title                  = EXCLUDED.title,
		    valid_from             = EXCLUDED.valid_from,
		    valid_until            = EXCLUDED.valid_until,
		    created_at             = EXCLUDED.created_at
	`
	_, err := w.pool.Exec(ctx, q,
		r.DID, r.RKey, r.VenueURI, r.RRule,
		r.SlotDurationMinutes, r.Title,
		r.ValidFrom, r.ValidUntil, r.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert recurrence %s/%s: %w", r.DID, r.RKey, err)
	}
	return nil
}

func (w *PgWriter) UpsertSlot(ctx context.Context, s Slot) error {
	// The slot owner (the venue admin's DID) is the only writer of
	// claimed_by_uri at the protocol level; the indexer mirrors that here.
	// Events written by *other* DIDs land in event_records and are joined
	// in via UpsertEvent's slots-update sub-statement.
	const q = `
		INSERT INTO slots (
		    did, rkey, venue_uri, recurrence_uri,
		    start_at, end_at, status, claimed_by_uri,
		    notes, created_at
		) VALUES (
		    $1, $2, $3, $4,
		    $5, $6, $7, $8,
		    $9, $10
		)
		ON CONFLICT (did, rkey) DO UPDATE SET
		    venue_uri       = EXCLUDED.venue_uri,
		    recurrence_uri  = EXCLUDED.recurrence_uri,
		    start_at        = EXCLUDED.start_at,
		    end_at          = EXCLUDED.end_at,
		    status          = EXCLUDED.status,
		    claimed_by_uri  = EXCLUDED.claimed_by_uri,
		    notes           = EXCLUDED.notes,
		    created_at      = EXCLUDED.created_at
	`
	_, err := w.pool.Exec(ctx, q,
		s.DID, s.RKey, s.VenueURI, s.RecurrenceURI,
		s.Start, s.End, s.Status, s.ClaimedByURI,
		s.Notes, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert slot %s/%s: %w", s.DID, s.RKey, err)
	}
	return nil
}

// UpsertEvent stores the event record and applies the W3 collision rule:
// when multiple events reference the same slot, the winner is the event
// with the *earliest* record createdAt. Slot.claimed_by_uri is updated only
// if the incoming event beats the existing claim. Retraction on
// out-of-order arrival is automatic — a later-arriving but earlier-created
// event will overwrite a current claim.
//
// This runs in a single transaction so the event row and the slot mutation
// are atomic.
func (w *PgWriter) UpsertEvent(ctx context.Context, e Event) error {
	if e.Tags == nil {
		e.Tags = []string{}
	}
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insertEvent = `
		INSERT INTO event_records (
		    did, rkey, slot_uri, title, description,
		    tags, visibility, capacity_cap, external_url, created_at
		) VALUES (
		    $1, $2, $3, $4, $5,
		    $6, $7, $8, $9, $10
		)
		ON CONFLICT (did, rkey) DO UPDATE SET
		    slot_uri      = EXCLUDED.slot_uri,
		    title         = EXCLUDED.title,
		    description   = EXCLUDED.description,
		    tags          = EXCLUDED.tags,
		    visibility    = EXCLUDED.visibility,
		    capacity_cap  = EXCLUDED.capacity_cap,
		    external_url  = EXCLUDED.external_url,
		    created_at    = EXCLUDED.created_at
	`
	if _, err := tx.Exec(ctx, insertEvent,
		e.DID, e.RKey, e.SlotURI, e.Title, e.Description,
		e.Tags, e.Visibility, e.CapacityCap, e.ExternalURL, e.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert event %s/%s: %w", e.DID, e.RKey, err)
	}

	// W3 collision rule: incoming event wins iff (a) no current claim, or
	// (b) its created_at is strictly earlier than the existing claimant's,
	// with (did, rkey) as the deterministic tiebreaker per docs/SPEC.md.
	// Re-asserting the same event's claim is idempotent.
	const eventURI = "at://%s/app.hekate.event/%s"
	const claimSlot = `
		UPDATE slots
		SET claimed_by_uri = $1, status = 'claimed'
		WHERE ('at://' || did || '/app.hekate.slot/' || rkey) = $2
		  AND (
		      claimed_by_uri IS NULL
		      OR claimed_by_uri = $1
		      OR EXISTS (
		          SELECT 1 FROM event_records existing
		          WHERE ('at://' || existing.did || '/app.hekate.event/' || existing.rkey) = slots.claimed_by_uri
		            AND (
		                existing.created_at > $3
		                OR (existing.created_at = $3 AND (existing.did, existing.rkey) > ($4, $5))
		            )
		      )
		  )
	`
	if _, err := tx.Exec(ctx, claimSlot,
		fmt.Sprintf(eventURI, e.DID, e.RKey),
		e.SlotURI,
		e.CreatedAt,
		e.DID, e.RKey,
	); err != nil {
		return fmt.Errorf("claim slot for event %s/%s: %w", e.DID, e.RKey, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (w *PgWriter) UpsertRSVP(ctx context.Context, r RSVP) error {
	const q = `
		INSERT INTO rsvps (
		    did, rkey, event_uri, status,
		    guest_count, note, created_at
		) VALUES (
		    $1, $2, $3, $4,
		    $5, $6, $7
		)
		ON CONFLICT (did, rkey) DO UPDATE SET
		    event_uri    = EXCLUDED.event_uri,
		    status       = EXCLUDED.status,
		    guest_count  = EXCLUDED.guest_count,
		    note         = EXCLUDED.note,
		    created_at   = EXCLUDED.created_at
	`
	_, err := w.pool.Exec(ctx, q,
		r.DID, r.RKey, r.EventURI, r.Status,
		r.GuestCount, r.Note, r.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert rsvp %s/%s: %w", r.DID, r.RKey, err)
	}
	return nil
}

// DeleteByKey removes a record across all typed tables for a given
// (did, collection, rkey) tombstone. Used by the indexer when the firehose
// emits a delete op.
func (w *PgWriter) DeleteByKey(ctx context.Context, did, collection, rkey string) error {
	var table string
	switch collection {
	case "app.hekate.venue":
		table = "venues"
	case "app.hekate.recurrence":
		table = "recurrences"
	case "app.hekate.slot":
		table = "slots"
	case "app.hekate.event":
		table = "event_records"
	case "app.hekate.rsvp":
		table = "rsvps"
	default:
		return fmt.Errorf("delete: unknown collection %q", collection)
	}
	// nolint:gosec // table is whitelisted above; did/rkey are parameters.
	q := fmt.Sprintf(`DELETE FROM %s WHERE did = $1 AND rkey = $2`, table)
	if _, err := w.pool.Exec(ctx, q, did, rkey); err != nil {
		return fmt.Errorf("delete %s/%s/%s: %w", did, collection, rkey, err)
	}
	return nil
}

