# Roadmap

Milestones are sized so each one produces a shippable artifact. This is a deliberate hedge against the user's apps-stall pattern (per `feedback_finish_apps`): if any later milestone never lands, the earlier ones still leave a useful protocol + indexer + CLI in the world.

## M-1: Smoke Signal coordination *(week 1, 2026-04-29 → 2026-05-06)*

Open a discourse/issue thread with [Smoke Signal](https://smokesignal.events/) maintainers. Decide alignment vs. divergence. Outcome recorded in `docs/adr/0002-smoke-signal-relationship.md`. Until this lands, the `event` and `rsvp` lexicons are subject to change.

## M0: Lexicon v0.1 *(by 2026-05-29 — 30 days)*

The durable artifact. Even if every other milestone slips, this one is real.

- [x] `lexicons/app/hekate/{venue,slot,recurrence,event,rsvp}.json`
- [x] `docs/SPEC.md` narrative spec
- [x] `docs/adr/0001-pin-atproto-sdk-versions.md`
- [ ] `make lex-validate` passes against upstream lexicon meta-schema
- [ ] First-pass review by ATProto-aware reviewer (community or maintainer)
- [ ] Tag `lexicon-v0.1.0`

## M1: Server v0.1 *(by 2026-10-29 — kill date)* — **[KILLED 2026-10-29]**

Kill criterion triggered on 2026-10-29. The firehose wiring and PgWriter were implemented (commits e4120a3, 7bde849) and the external-PDS round-trip integration test was written (server/internal/indexer/indexer_integration_test.go, commit added 2026-07-01), but the test requires live PDS credentials that were never provided to CI, and no commit marks the "Reads at least one external PDS's records end-to-end" deliverable as verified. All M1 ROADMAP checkboxes remain unchecked at the kill date. See ADR 0099.

- [ ] Firehose subscriber (filtered to `app.hekate.*`)
- [ ] Indexer: events table → typed tables, idempotent
- [ ] Recurrence expander: 90-day horizon, daily tick
- [ ] Read API: `/venues`, `/slots`, `/events` with bbox + radius + filters
- [ ] PostGIS migrations + sqlc-generated queries
- [ ] Integration tests (testcontainers-go) green
- [ ] Self-host smoke green
- [ ] **Reads at least one external PDS's records end-to-end**

**Kill criterion (precommitted):** if M1 has not shipped reads-from-external-PDS by **2026-10-29**, scope down to lexicons-only and pause. This is not negotiable mid-flight. **Triggered 2026-10-29 — see `docs/adr/0099-kill-criterion-2026-10-29.md`.**

## M2: CLI v0.1 *(concurrent with M1)* — **[OUT OF SCOPE — paused per kill criterion]**

The v0.1 reference client. Promoted from "stall insurance" to first-class per the salt-mines panel.

- [ ] `hekate venue create | list | get`
- [ ] `hekate slot post | list | cancel`
- [ ] `hekate event claim | list`
- [ ] `hekate rsvp going | maybe | declined`
- [ ] OAuth login flow (`@atproto/oauth-client-go` equivalent or manual app password)
- [ ] Round-trip integration test green
- [ ] Single-binary release artifacts (linux/amd64, linux/arm64, darwin/arm64)

## M2.5: AT Protocol grant submission *(by 2026-06-29 — 60 days)* — **[OUT OF SCOPE — paused per kill criterion]**

Per the salt-mines panel: only realistic monetization path. Forces a written external-facing scope artifact even if the grant is declined.

- [ ] Application drafted referencing M0 deliverables
- [ ] Submitted via the active grants channel (TBD)
- [ ] Outcome recorded in `docs/adr/0003-grant-decision.md`

## M3: Web v0.1 *(post-M1, off the critical path)* — **[OUT OF SCOPE — paused per kill criterion]**

Only pursued after M1 ships *and* there is an external user actually reading PDS records.

- [ ] SvelteKit 2 + Svelte 5 with `@sveltejs/adapter-node`
- [ ] OAuth via `@atproto/oauth-client-browser`
- [ ] Routes: `/`, `/v/[handle]`, `/s/[atUri]`, `/e/[atUri]`, `/me`
- [ ] MapLibre GL for venue map
- [ ] PWA manifest + service worker
- [ ] Reads via hekate-server REST; writes via `@atproto/api` to user PDS
- [ ] `pnpm check && pnpm build` green

## M4: Public reference instance *(post-M3)* — **[OUT OF SCOPE — paused per kill criterion]**

A live deployment at `hekate.<domain>` (domain TBD pending the GoDaddy → Cloudflare migration).

- [ ] DNS + TLS terminator
- [ ] `lexicons.hekate.app` resolves the `$type` URIs
- [ ] At least one non-author user successfully publishes and indexes records
- [ ] Public README updated with the reference instance URL

## M5: Native mobile *(optional, only if adoption justifies)* — **[OUT OF SCOPE — paused per kill criterion]**

Flutter or native — **not** scoped now. Revisit after M4 ships.

## What's *deliberately* out of scope

- Ticketing, payments, calendar invites (use external links via `event.externalUrl`)
- Real-time chat / comments (use existing ATProto patterns)
- Analytics / popularity ranking (the indexer is read; aggregation is a separate, downstream concern)
- Multi-language i18n in v0.1 (English only; structure permits later)
