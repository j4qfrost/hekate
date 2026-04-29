-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS postgis;
-- +goose StatementEnd

-- Raw firehose log. Backpressure lives here: unprocessed rows accumulate
-- when the indexer is slow but the firehose connection is preserved.
CREATE TABLE events (
    seq             BIGSERIAL PRIMARY KEY,
    did             TEXT NOT NULL,
    collection      TEXT NOT NULL,
    rkey            TEXT NOT NULL,
    cid             TEXT NOT NULL,
    record          BYTEA NOT NULL,
    record_created_at TIMESTAMPTZ NOT NULL,
    observed_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at    TIMESTAMPTZ
);
CREATE INDEX events_unprocessed_idx ON events (observed_at) WHERE processed_at IS NULL;
CREATE UNIQUE INDEX events_record_idx ON events (did, collection, rkey, cid);

-- Materialised venues table. Geo is a PostGIS point; SRID 4326 (WGS 84).
CREATE TABLE venues (
    did              TEXT NOT NULL,
    rkey             TEXT NOT NULL,
    name             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    geo              GEOGRAPHY(POINT, 4326) NOT NULL,
    altitude_meters  DOUBLE PRECISION,
    address_text     TEXT NOT NULL DEFAULT '',
    locality         TEXT NOT NULL DEFAULT '',
    region           TEXT NOT NULL DEFAULT '',
    country          CHAR(2) NOT NULL DEFAULT '',
    postal_code      TEXT NOT NULL DEFAULT '',
    capacity         INTEGER,
    amenities        TEXT[] NOT NULL DEFAULT '{}',
    booking_policy   TEXT NOT NULL CHECK (booking_policy IN ('open', 'review')),
    contact_email    TEXT NOT NULL DEFAULT '',
    contact_url      TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (did, rkey)
);
CREATE INDEX venues_geo_idx ON venues USING GIST (geo);
CREATE INDEX venues_amenities_idx ON venues USING GIN (amenities);

CREATE TABLE recurrences (
    did                    TEXT NOT NULL,
    rkey                   TEXT NOT NULL,
    venue_uri              TEXT NOT NULL,
    rrule                  TEXT NOT NULL,
    slot_duration_minutes  INTEGER NOT NULL CHECK (slot_duration_minutes BETWEEN 5 AND 10080),
    title                  TEXT NOT NULL DEFAULT '',
    valid_from             TIMESTAMPTZ NOT NULL,
    valid_until            TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (did, rkey)
);
CREATE INDEX recurrences_venue_idx ON recurrences (venue_uri);

CREATE TABLE slots (
    did             TEXT NOT NULL,
    rkey            TEXT NOT NULL,
    venue_uri       TEXT NOT NULL,
    recurrence_uri  TEXT,
    start_at        TIMESTAMPTZ NOT NULL,
    end_at          TIMESTAMPTZ NOT NULL CHECK (end_at > start_at),
    status          TEXT NOT NULL CHECK (status IN ('open', 'claimed', 'cancelled')),
    claimed_by_uri  TEXT,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (did, rkey)
);
CREATE INDEX slots_venue_idx ON slots (venue_uri);
CREATE INDEX slots_window_idx ON slots (start_at, end_at) WHERE status = 'open';
CREATE UNIQUE INDEX slots_recurrence_start_idx ON slots (recurrence_uri, start_at) WHERE recurrence_uri IS NOT NULL;

CREATE TABLE event_records (
    did             TEXT NOT NULL,
    rkey            TEXT NOT NULL,
    slot_uri        TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    visibility      TEXT NOT NULL CHECK (visibility IN ('public', 'unlisted')),
    capacity_cap    INTEGER,
    external_url    TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (did, rkey)
);
CREATE INDEX event_records_slot_idx ON event_records (slot_uri);
CREATE INDEX event_records_organizer_idx ON event_records (did, created_at);

CREATE TABLE rsvps (
    did          TEXT NOT NULL,
    rkey         TEXT NOT NULL,
    event_uri    TEXT NOT NULL,
    status       TEXT NOT NULL CHECK (status IN ('going', 'maybe', 'declined')),
    guest_count  INTEGER NOT NULL DEFAULT 0,
    note         TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (did, rkey)
);
CREATE INDEX rsvps_event_idx ON rsvps (event_uri);
CREATE INDEX rsvps_attendee_idx ON rsvps (did, event_uri, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS rsvps;
DROP TABLE IF EXISTS event_records;
DROP TABLE IF EXISTS slots;
DROP TABLE IF EXISTS recurrences;
DROP TABLE IF EXISTS venues;
DROP TABLE IF EXISTS events;
-- Leave PostGIS extension installed; dropping it would affect other schemas.
