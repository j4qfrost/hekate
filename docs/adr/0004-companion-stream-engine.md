# ADR 0004 — Companion stream engine: RisingWave

- Status: Accepted
- Date: 2026-04-29
- Deciders: hekate maintainers

## Context

`docs/ARCHITECTURE.md` commits hekate to a Postgres-only core: three goroutines (firehose, indexer, recurrence-expander) communicating exclusively through tables, with `make selfhost-smoke` enforcing a "5-minute self-host" promise. That core handles geographic queries (PostGIS), temporal queries (B-tree indices on slot windows), and identity-axis lookups well.

It does **not** handle the slot-collision rule from `docs/SPEC.md` cleanly. The spec says: when multiple `app.hekate.event` records claim the same `open` slot, the winner is the event with the earliest `record.createdAt`, regardless of arrival order at the indexer. Because the firehose is unordered relative to record-time, a later-arriving event can win retroactively, requiring the indexer to retract a previously-emitted answer. Expressing that pattern with batch SQL + Postgres triggers is awkward and error-prone — it is the canonical "out-of-order keyed reduction" workload that stream-processing engines exist to solve.

Per project `CLAUDE.md`: introducing any heavy-infra component (search/cache layer) requires an ADR. A stream engine qualifies.

## Decision

Adopt **RisingWave** as the companion stream-processing engine, in single-image standalone (`playground`) mode, consuming Postgres CDC against a dedicated insert-only projection table (`events_decoded`).

Specifically:

1. **Topology.** Go indexer goroutine writes raw events to `events` (BYTEA CBOR) and decoded rows to `events_decoded` (typed columns, sparse per-collection). Postgres logical-replication publishes `events_decoded` only, INSERT-only (`WITH (publish = 'insert')`). RisingWave's Postgres-CDC source consumes that publication. Materialised views over the CDC table feed the read API.
2. **Decoding stays in Go.** RisingWave never parses ATProto CBOR. Lexicon decoding lives in `server/internal/lexicon/` (generated, per ADR 0001) and the indexer goroutine does the work before writing to `events_decoded`.
3. **Companion stream is observational, not authoritative.** The Go indexer remains the sole writer of `slots.claimed_by_uri` and the other typed tables. RisingWave's views are read paths only; the engine never feeds back into the source-of-truth tables.
4. **Workload scope at v0.1.** Two materialised views: `w1_occupancy` (per-venue hourly slot counts; smoke) and `w3_collision` (per-slot earliest-`record_created_at` winner; the workload that justifies the engine). The RSVP-funnel workload originally considered is dropped — the `app.hekate.rsvp` lexicon is M-1-pending (ADR 0002) and the workload would not differentiate engine choice anyway.
5. **Compose isolation.** RisingWave lives in a separate stack at `deploy/companion/docker-compose.yml`, brought up via `make companion-up`. It is **not** added to `deploy/docker-compose.yml` until the read API can consume the views; the smoke target's footprint is preserved. Production-compose integration is a follow-up decision after M1's read API surfaces a concrete consumer of `w3_collision`.

### Alternatives considered

- **Apache Flink.** Mature watermark/event-time semantics handle W3 cleanly. Rejected on operational footprint: JobManager + TaskManager + state backend + connector init container roughly doubles compose memory and triples the service count vs. RisingWave's single image. The 5-minute self-host promise is load-bearing for the project's salt-mines verdict ("credibility play"); Flink's compose surface erodes it.
- **Postgres triggers / scheduled batch.** Considered for the "neither, YAGNI" outcome. Rejected because the out-of-order retraction case in W3 forces either re-running batch jobs over windowed data (latency-poor) or hand-maintained trigger logic that re-derives prior winners on every late insert (correctness-poor). Both reproduce a worse stream engine with no operational savings once the projection table exists anyway.
- **Kafka/Redpanda in front of the engine.** Rejected: violates the architectural invariant that all in-band coordination flows through Postgres, and adds a third stateful service to compose — strictly worse than CDC for self-host.

## Consequences

- **New migration shape.** `server/migrations/0002_companion_projection.sql` adds `events_decoded` and the `hekate_companion` publication. The indexer (and, for now, the fixture loader at `companion/fixturegen/`) writes both rows in a single transaction — the FK from `events_decoded.seq → events.seq` keeps the projection an honest mirror of raw arrivals.
- **Postgres flag requirement.** `wal_level=logical` plus `max_replication_slots`/`max_wal_senders` are required at startup. `deploy/postgres-init/logical-replication.md` documents this; the companion compose's Postgres service runs with these flags via `command:`. Production compose is unchanged until production-compose integration.
- **Spike artifacts in tree.** `companion/fixturegen/` (Go), `companion/sql/{01_source,02_w1_occupancy,03_w3_collision}.sql`, `companion/verify/` (W3 correctness against a Postgres-side ground truth) all live at the workspace root level rather than under `server/` or `cli/`, to keep the spike-shaped code from bleeding into subprojects with their own toolchains and tests.
- **Pinning policy.** RisingWave version is pinned in `deploy/companion/docker-compose.yml` (`risingwavelabs/risingwave:v2.1.0`). Connector and SDK upgrades follow ADR 0001's discrete-PR discipline.
- **Lexicon-pending caveat.** Workload SQL refers only to slot AT-URIs, `did`, `rkey`, and `record_created_at` — fields whose shape is stable. It does not depend on `event` or `rsvp` body fields, both of which are M-1-pending per ADR 0002.
- **Observability deferred.** RisingWave's dashboard (port 5691) is exposed in compose for spike use; productionising metrics + alerts is a separate decision when the engine joins the production stack.

## References

- `docs/SPEC.md` — collision-resolution rule (the W3 spec).
- `docs/ARCHITECTURE.md` — three-goroutine core; the Postgres-only invariant the companion preserves.
- `docs/SELFHOST.md` — 5-minute promise the production compose protects.
- `docs/adr/0001-pin-atproto-sdk-versions.md` — pinning discipline applied here for the RisingWave image.
- `docs/adr/0002-smoke-signal-relationship.md` — M-1-pending lexicons constraining workload scope.
- RisingWave Postgres-CDC: https://docs.risingwave.com/integrations/sources/postgres-cdc
