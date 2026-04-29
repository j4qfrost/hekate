-- RisingWave: Postgres-CDC source over the hekate_companion publication.
--
-- The publication is INSERT-ONLY (see server/migrations/0002), which keeps
-- this stream free of indexer-side processed_at UPDATE noise.
--
-- Connection params come from environment defaults baked into the companion
-- compose stack; override via psql variables if running by hand.
CREATE SOURCE IF NOT EXISTS pg_hekate WITH (
    connector       = 'postgres-cdc',
    hostname        = 'postgres',
    port            = '5432',
    username        = 'hekate',
    password        = 'hekate',
    database.name   = 'hekate',
    schema.name     = 'public',
    publication.name = 'hekate_companion',
    slot.name       = 'hekate_companion_slot'
);

-- The CDC-attached table is RisingWave's view onto Postgres's events_decoded.
-- Type widths mirror the migration; primary key matches.
CREATE TABLE IF NOT EXISTS events_decoded (
    seq               BIGINT,
    did               VARCHAR,
    collection        VARCHAR,
    rkey              VARCHAR,
    record_created_at TIMESTAMPTZ,
    observed_at       TIMESTAMPTZ,
    venue_uri         VARCHAR,
    slot_uri          VARCHAR,
    start_at          TIMESTAMPTZ,
    end_at            TIMESTAMPTZ,
    status            VARCHAR,
    title             VARCHAR,
    PRIMARY KEY (seq)
)
FROM pg_hekate TABLE 'public.events_decoded';
