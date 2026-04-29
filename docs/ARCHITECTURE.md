# Architecture

## Components

```
                          ATProto firehose (relay)
                                  │
                                  ▼
        ┌────────────────────────────────────────────────────┐
        │                  hekate-server (Go)                 │
        │                                                     │
        │   ┌──────────┐    ┌──────────┐    ┌──────────┐    │
        │   │ firehose │───▶│ indexer  │───▶│  read    │    │
        │   │ consumer │    │ (postgis)│    │  API     │    │
        │   └──────────┘    └──────────┘    └──────────┘    │
        │                         ▲              │           │
        │                         │              │           │
        │                  ┌──────────────┐      │           │
        │                  │ recurrence   │      │           │
        │                  │ expander     │      │           │
        │                  └──────────────┘      │           │
        └─────────────────────────────────────────┼───────────┘
                                                  │
              ┌───────────────────────────────────┴─────────────┐
              │                                                  │
         ┌────▼────┐                                       ┌────▼────┐
         │ hekate  │                                       │  web    │
         │  CLI    │                                       │ (Svelte │
         │ (Go)    │                                       │  Kit)   │
         └────┬────┘                                       └────┬────┘
              │ writes records                                  │ writes records
              ▼                                                  ▼
         ┌──────────────────────────────────────────────────────────┐
         │                  user/venue PDS                           │
         │              (app.hekate.* records)                       │
         └──────────────────────────────────────────────────────────┘
```

The server is a **read-only AppView**. It does not own user data; it indexes records from PDSes for query.

## Why three goroutines?

1. **Firehose consumer.** Maintains the WebSocket connection to a relay (default `wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos`), filters for `app.hekate.*` collections, and writes raw events to a `events` table in Postgres. Backpressure is handled by `events.processed_at` — unprocessed events accumulate when the indexer is slow but the firehose connection is preserved.
2. **Indexer.** Consumes `events.processed_at IS NULL` rows in `createdAt` order, validates them against the lexicon spec rules, and materialises typed rows in `venues`, `slots`, `events`, `rsvps`. Idempotent on the `(authoring_did, rkey)` primary key.
3. **Recurrence expander.** A daily tick. For each `recurrences` row, materialise the next 90 days of derived `slots` rows (status=open). Re-running is idempotent: existing slots are skipped by their `(recurrence_uri, start)` unique index.

These three communicate only through Postgres tables. No in-process channels. This makes the server crash-safe and horizontally scalable: multiple replicas can share the same Postgres without coordination beyond row-level advisory locks (which we'll add only if we need them).

## Why Postgres + PostGIS?

- Geographic queries (`venue near point`, `venues in bbox`) are first-class for venue-first discovery. SQLite with R-Tree could work but PostGIS gives us GIST indexes, ST_DWithin with metric distance, and an upgrade path to spatial joins.
- We already need transactions for the `events → venues/slots/events/rsvps` materialisation pipeline; SQL is the right tool.
- Self-hosters get a single Postgres container; no extra search/cache layer to operate.

## Why the CLI is the v0.1 reference client

Per the salt-mines panel evaluation (2026-04-29), the user's historical apps-stall pattern (per `feedback_finish_apps`) makes UI work the highest-risk milestone. The CLI:

- Has zero stall risk: complete in ~10 commands, no design, no theming, no responsive layout.
- Demonstrates the protocol completely: any record any client could write, the CLI can write.
- Survives every failure mode of the UI work: even if M3 (web) never lands, the CLI makes the indexer useful.
- Acts as the contract: the CLI's commands are the smoke test for the read API.

The web client is M3 and explicitly off the critical path.

## Federation

There is no central server. Anyone can:
- Run a hekate-server pointed at any relay.
- Index any DID's records by configuring a different DID-set or by accepting all `app.hekate.*` records from the firehose.
- Publish `app.hekate.*` records from any PDS (Bluesky-hosted or self-hosted).

The reference instance (planned at M4) is **a peer**, not an authority.

## Open questions

- **Lexicon publication URL.** `lexicons.hekate.app` (or similar) needs to resolve `$type` URIs. Tied to the GoDaddy → Cloudflare migration in flight (per `project_godaddy_migration` memory).
- **Push notifications.** Web push works but requires a VAPID key per instance. Deferred to post-M4.
- **Rate limits.** None at the protocol level; per-instance throttling lives in the indexer config.
