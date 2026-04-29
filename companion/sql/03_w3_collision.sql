-- W3 — Slot conflict resolution (the decisive workload).
--
-- docs/SPEC.md §"Collision resolution": when multiple events claim the same
-- slot, the winner is the event with the earliest record_created_at. The
-- fixture loader's --scenario=skewed mode emits events whose observed_at is
-- later than their record_created_at, so a later-arriving event can win
-- retroactively — RisingWave must retract the previous winner and emit the
-- new one.
--
-- Tiebreaker (did, rkey) keeps the result deterministic if two competing
-- events ever share an exact record_created_at. The fixture loader spaces
-- competitors apart so the tiebreaker should not fire under normal runs.
CREATE MATERIALIZED VIEW IF NOT EXISTS w3_collision AS
SELECT
    slot_uri,
    did            AS winner_did,
    rkey           AS winner_rkey,
    record_created_at AS winner_created_at
FROM (
    SELECT
        slot_uri,
        did,
        rkey,
        record_created_at,
        ROW_NUMBER() OVER (
            PARTITION BY slot_uri
            ORDER BY record_created_at, did, rkey
        ) AS rn
    FROM events_decoded
    WHERE collection = 'app.hekate.event'
      AND slot_uri IS NOT NULL
) ranked
WHERE rn = 1;
