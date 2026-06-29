# AT Protocol Grant Application — hekate

**Project:** hekate — venue-first event coordination on AT Protocol
**Repository:** https://github.com/j4qfrost/hekate (Apache-2.0)
**Maintainer:** j4qfrost
**Application date:** 2026-06-29
**Status:** DRAFT — to be reviewed and submitted by maintainer

---

## Project Summary

hekate is a venue-first event coordination protocol built as a lexicon extension for the AT Protocol. Where existing AT Protocol events applications (notably Smoke Signal) are event-first — a user publishes an event and attaches a venue description inline — hekate treats venues as independent, first-class, addressable AT Protocol records. A venue with no events is still useful: it exists on the network as a discoverable entity with a geographic position, a capacity, and a booking policy. Organizers claim discrete *slots* that venue owners publish; RSVPs attach to the resulting event. This inversion — venue → slot → event → RSVP — enables a geographic slot marketplace that the current AT Protocol lexicon landscape does not address.

The five lexicons under `app.hekate.*` (`venue`, `slot`, `recurrence`, `event`, `rsvp`) are the primary artifact. They are designed to be indexable by any AppView implementation: the hekate reference indexer (Go, backed by Postgres + PostGIS) is one such implementation, and nothing in the lexicon design prevents others. The lexicons deliberately avoid encoding anything that requires a central authority — there is no registry of venues, no reservation system, no central event database. The indexer is a read-only AppView: it observes the AT Protocol firehose, materialises derived state, and serves geographic and temporal queries. Clients write directly to their own PDS.

The project is open-source (Apache-2.0), public from day one, and designed to interoperate with the existing AT Protocol ecosystem rather than compete with it. The `venue`, `slot`, and `recurrence` lexicons are novel contributions with no prior art in the AT Protocol lexicon namespace. The `event` and `rsvp` lexicons are deliberately scoped to the slot-claim relationship so they complement rather than duplicate Smoke Signal's event/RSVP primitives (see *Coordination with Smoke Signal* below).

---

## Deliverables to Date

### M0: Five lexicons under `app.hekate.*` (shipped)

| Lexicon | File | Description |
|---|---|---|
| `app.hekate.venue` | `lexicons/app/hekate/venue.json` | Venue record: name, geo, capacity, booking policy |
| `app.hekate.slot` | `lexicons/app/hekate/slot.json` | Bookable time window; slot state machine (open/claimed/cancelled) |
| `app.hekate.recurrence` | `lexicons/app/hekate/recurrence.json` | RFC 5545 RRULE template that materialises into slots |
| `app.hekate.event` | `lexicons/app/hekate/event.json` | Slot-claim record owned by an organizer DID |
| `app.hekate.rsvp` | `lexicons/app/hekate/rsvp.json` | Attendee response to an event (going/maybe/declined) |

The lexicons are accompanied by a narrative specification (`docs/SPEC.md`) that documents the state machine, validation rules enforced by the indexer (not the PDS), collision resolution semantics, and the discovery model (geographic / temporal / identity axes). The spec is normative relative to the JSON; a lexicon validator script (`scripts/lex_validate.py`) runs on CI.

Coordinate encoding was corrected on 2026-05-20 (venue geo lat/lon encoded as strings, not floats) after the initial wire was laid — an example of the iterate-in-public approach that the project intends to continue.

### M1: Reference indexer — Go server (in progress; firehose + indexer pipeline landed)

The Go server (`server/`) implements the three-goroutine architecture documented in `docs/ARCHITECTURE.md`:

- **Firehose consumer** (`server/internal/firehose/`): WebSocket subscriber to the AT Protocol relay (`wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos`), filtered to `app.hekate.*` collections. Backed by indigo (the AT Protocol Go SDK, pinned per `docs/adr/0001-pin-atproto-sdk-versions.md`). Landed 2026-04-29.
- **Indexer / PgWriter** (`server/internal/indexer/`): Consumes raw firehose events, decodes CBOR, validates against lexicon rules, materialises typed rows. Includes the W3 collision rule (earliest `record.createdAt` wins on slot-claim races). Landed 2026-05-18–20.
- **Database migrations** (`server/migrations/`): Two goose migrations — `0001_init.sql` (venues, slots, events, rsvps, PostGIS) and `0002_companion_projection.sql` (the decoded projection table for RisingWave CDC).

Still pending for M1: recurrence expander (90-day horizon daily tick), read API (`/venues`, `/slots`, `/events` with geo filters), integration tests, and the key milestone gate: *reads at least one external PDS's records end-to-end* (kill date 2026-10-29).

### Companion stream engine (ADR 0004, shipped as spike)

The slot-collision rule ("earliest `createdAt` wins regardless of firehose arrival order") is a canonical out-of-order keyed reduction workload. hekate adopted RisingWave as a companion stream engine (`companion/`, `deploy/companion/`) to handle this correctly:

- **`companion/sql/03_w3_collision.sql`** — the W3 materialised view: per-slot earliest-`record_created_at` winner.
- **`companion/sql/02_w1_occupancy.sql`** — per-venue hourly slot occupancy counts.
- **`companion/fixturegen/`** — Go fixture generator for spike testing.
- **`companion/verify/`** — W3 correctness verifier against a Postgres-side ground truth.

The companion is isolated from the production compose (`deploy/docker-compose.yml`) until the read API surfaces a concrete consumer of `w3_collision`. Self-host smoke is unaffected.

### Observability stack (ADR 0005, shipped)

The server is instrumented with OpenTelemetry (OTLP/gRPC) and ships a pre-wired LGTM stack (Loki + Grafana + Tempo + Mimir) for local and self-host use:

- `server/internal/telemetry/` — OTel SDK init + hekate-specific metrics (`hekate_firehose_events_total`, `hekate_indexer_records_indexed_total`, etc.)
- `deploy/observability/docker-compose.yml` — LGTM all-in-one compose (opt-in via `make obs-up`)
- `deploy/observability/grafana/dashboards/hekate-overview.json` — pre-provisioned Grafana dashboard

Observability is off by default in the production compose; exporters no-op when `HEKATE_OTLP_ENDPOINT` is unset.

### Architecture decisions on record

| ADR | Title | Status |
|---|---|---|
| `docs/adr/0001-pin-atproto-sdk-versions.md` | Pin AT Protocol SDK versions; track Lexicon SDK redesign | Accepted |
| `docs/adr/0002-smoke-signal-relationship.md` | Relationship with Smoke Signal | Pending coordination |
| `docs/adr/0003-grant-decision.md` | AT Protocol grant submission outcome | Draft (this application) |
| `docs/adr/0004-companion-stream-engine.md` | Companion stream engine: RisingWave | Accepted |
| `docs/adr/0005-lgtm-observability-stack.md` | LGTM observability stack via OpenTelemetry | Accepted |

### Go CLI (`cli/`) — stub stage

The CLI (`cli/`) is scaffolded with command structure (`cmd/`, `internal/`). It is the planned v0.1 reference client for publishing `app.hekate.*` records (venue create/list, slot post/list/cancel, event claim/list, rsvp going/maybe/declined) with an OAuth login flow. Currently at stubs; implementation is concurrent with M1.

---

## What Grant Funding Would Unlock

The lexicons and indexer are being built on existing maintainer time. Grant funding would accelerate three specific areas that are currently blocked or deferred:

### 1. M3: SvelteKit web client (`web/`, currently off the critical path)

The web client exists as scaffolding (`web/`, SvelteKit 2 + Svelte 5) but is explicitly off the critical path until M1 ships. It is the highest-value surface for protocol adoption: a live map of venues and slots makes the protocol legible to a non-technical audience. Funding would allow sustained work on the web client in parallel with M1 rather than sequentially after it.

Concrete deliverables: MapLibre GL venue map, OAuth via `@atproto/oauth-client-browser`, routes `/v/[handle]` (venue detail) and `/s/[atUri]` (slot detail), PWA manifest. All writes go through the user's PDS via `@atproto/api`; the web client is a pure AppView consumer.

### 2. M4: Public reference instance at `hekate.app`

The reference instance is a peer — not an authority — that demonstrates federation. At least one non-maintainer user successfully publishing and indexing records against a running hekate-server would constitute the milestone. Costs covered: hosting, DNS + TLS termination (GoDaddy → Cloudflare migration in flight), lexicon URL resolution at `lexicons.hekate.app`.

### 3. Lexicon SDK migration (ADR 0001 follow-up)

The AT Protocol Spring 2026 roadmap introduces a Buf-inspired Lexicon SDK redesign. hekate's generated code under `server/internal/lexicon/` (committed per ADR 0001) will need migration as a discrete PR when the new codegen ships in indigo. Doing this migration promptly (rather than letting the pin stagnate) is a visible maintenance signal and keeps hekate's lexicons interoperable with new SDK-generated clients. ADR 0001 documents the discrete-PR discipline; funding buys the time to do it right.

---

## What This Is Not

hekate is a **protocol contribution**, not a product business. This application does not request funding for:

- **Ticketing or payments.** Out of scope by design. `event.externalUrl` is the hook for external ticketing systems.
- **Real-time chat or comments.** The AT Protocol already has `app.bsky.feed.post` with replies; hekate does not reinvent it.
- **A centralised venue registry.** Centralised allowlists would contradict the federation model. Any PDS can host `app.hekate.venue` records.
- **Calendar invite generation.** iCal/ICS export is a thin adapter; useful but not a lexicon concern.
- **Analytics, ranking, or aggregation.** The indexer is a read path; popularity scoring is a downstream, opt-in concern for indexer operators.
- **Maintainer salary or infrastructure at scale.** The ask is scoped to protocol artifacts and the reference instance.

---

## Coordination with Smoke Signal

[Smoke Signal](https://smokesignal.events/) (Rust, Nick Gerakines et al.) is the most established AT Protocol events application. hekate's relationship with Smoke Signal is documented in `docs/adr/0002-smoke-signal-relationship.md`.

A coordination thread was opened with the Smoke Signal maintainers on 2026-05-19 (documented at `docs/outreach/smoke-signal-2026-05-coordination.md`). The options under consideration were:

- **A. Coexist (divergent specs):** hekate ships all five lexicons; Smoke Signal keeps its lexicons; indexers from each project observe the other's records independently.
- **B. hekate-as-venue-only:** hekate ships only `venue`, `slot`, and `recurrence`; event and RSVP records delegate to Smoke Signal's lexicon.
- **C. Joint standard:** Co-author a shared namespace (e.g. `community.lexicon.event.*`) that both projects implement.

The two-week coordination window closed 2026-06-02 without a formal response. Per ADR 0002's precommitted rule, hekate defaults to **option A (coexist)** with a public record that coordination was attempted. This outcome is not adversarial: the venue-first model and the event-first model are complementary, and a hekate indexer can in principle surface Smoke Signal events that reference hekate slots via `externalUrl` without any protocol change.

The AT Protocol grants program is well positioned to facilitate a deeper alignment conversation between the two projects if that is of interest to the program.

---

## Grant application checklist (for maintainer before submission)

- [ ] Verify the active grants channel / submission form URL (check https://atproto.com/ and recent Bluesky protocol posts)
- [ ] Add ADR 0002 resolution once Smoke Signal coordination is formally recorded
- [ ] Add a quote from at least one external AT Protocol ecosystem participant (optional but strengthens credibility)
- [ ] Verify lexicon validator CI passes (`make lex-validate`)
- [ ] Record submission link in `docs/adr/0003-grant-decision.md` after submitting

---

*This document was drafted on 2026-06-29 (M2.5 deadline) from the current repo state. It is a draft for maintainer review — do not submit without reviewing the checklist above.*
