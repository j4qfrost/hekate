-- W1 — Per-venue hourly slot occupancy.
--
-- Smoke workload: confirms the CDC path, schema mapping, and that materialised
-- views are queryable from outside the engine. A trivial GROUP BY that both
-- engines handle effortlessly; its value is in being unambiguous to verify
-- against a Postgres-side ground-truth recomputation.
CREATE MATERIALIZED VIEW IF NOT EXISTS w1_occupancy AS
SELECT
    venue_uri,
    date_trunc('hour', start_at) AS bucket_hour,
    status,
    COUNT(*) AS slot_count
FROM events_decoded
WHERE collection = 'app.hekate.slot'
  AND venue_uri IS NOT NULL
  AND start_at IS NOT NULL
GROUP BY venue_uri, date_trunc('hour', start_at), status;
