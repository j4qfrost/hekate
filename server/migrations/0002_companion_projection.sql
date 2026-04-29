-- +goose Up
-- +goose StatementBegin
-- Insert-only typed projection of events used as the upstream of the companion
-- stream-processing engine (RisingWave). Decoding lives in Go (the indexer
-- goroutine and the spike fixture loader); RisingWave never sees raw CBOR.
--
-- The projection is insert-only by design: a CDC publication that ships
-- UPDATEs would forward the indexer's processed_at writes on `events`, which
-- are noise to the companion engine. Mirroring the rule here keeps the
-- companion publication clean even if events_decoded later gains updateable
-- columns.
CREATE TABLE events_decoded (
    seq               BIGINT PRIMARY KEY REFERENCES events(seq) ON DELETE CASCADE,
    did               TEXT NOT NULL,
    collection        TEXT NOT NULL,
    rkey              TEXT NOT NULL,
    record_created_at TIMESTAMPTZ NOT NULL,
    observed_at       TIMESTAMPTZ NOT NULL,
    -- Sparse per-collection columns. NULL when the collection does not define
    -- the field. Workload SQL filters by collection before reading these.
    venue_uri         TEXT,
    slot_uri          TEXT,
    start_at          TIMESTAMPTZ,
    end_at            TIMESTAMPTZ,
    status            TEXT,
    title             TEXT
);
CREATE INDEX events_decoded_collection_idx ON events_decoded (collection, record_created_at);
CREATE INDEX events_decoded_slot_uri_idx ON events_decoded (slot_uri) WHERE slot_uri IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE PUBLICATION hekate_companion FOR TABLE events_decoded WITH (publish = 'insert');
-- +goose StatementEnd

-- +goose Down
DROP PUBLICATION IF EXISTS hekate_companion;
DROP TABLE IF EXISTS events_decoded;
