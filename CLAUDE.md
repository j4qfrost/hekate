# CLAUDE.md — hekate

This file is loaded by Claude Code (claude.ai/code) when working in this repository. It is the per-project counterpart to the workspace map at `~/Projects/CLAUDE.md`.

## What this project is

hekate is a venue-first event coordination protocol and reference implementation on the AT Protocol. Five lexicons under `app.hekate.*`, a Go indexer/API, a Go CLI (the v0.1 reference client), and a SvelteKit 2 web client (off the critical path).

The plan-of-record is `~/.claude/plans/create-project-that-helps-streamed-platypus.md` (rev 3 as of 2026-04-29). The roadmap is `docs/ROADMAP.md`. Architecture is `docs/ARCHITECTURE.md`. Lexicon spec is `docs/SPEC.md`. ADRs are `docs/adr/`.

## Critical context for any task

1. **Salt-mines verdict (2026-04-29):** *credibility play, not a cash project.* The lexicons + indexer are the durable artifacts. Apps stall — the user has historical pattern (per `feedback_finish_apps`). The web client is M3 and explicitly **off the critical path**; the Go CLI is the v0.1 reference client.

2. **Kill criterion (precommitted):** if M1 hasn't shipped reads-from-external-PDS by **2026-10-29**, scope down to lexicons-only and pause. Do not push past this date in scope. Surface it explicitly when planning subsequent work.

3. **M-1 dependency:** the `event` and `rsvp` lexicons are *subject to change* until coordination with [Smoke Signal](https://smokesignal.events/) completes. See `docs/adr/0002-smoke-signal-relationship.md`. Don't treat them as frozen yet.

4. **SDK pinning policy:** indigo (Go) and `@atproto/api` / `@atproto/lexicon` (TS) are pinned in lockfiles. The Buf-inspired Lexicon SDK redesign is incoming; migrate in discrete PRs only. See `docs/adr/0001-pin-atproto-sdk-versions.md`.

## Subproject toolchains (each treated independently)

| Path | Toolchain | Build/test |
|---|---|---|
| `lexicons/` | JSON + meta-schema validation | `make lex-validate` |
| `server/` | Go 1.22+, sqlc, goose, Postgres+PostGIS | `cd server && go test ./...` |
| `cli/` | Go 1.22+ | `cd cli && go test ./...` |
| `web/` | Node 22+, pnpm 9+, SvelteKit 2 | `cd web && pnpm check && pnpm build` |
| `deploy/` | Docker Compose | `make selfhost-smoke` |

Top-level `Makefile` orchestrates: `make dev`, `make test`, `make lex-validate`, `make selfhost-smoke`.

## Conventions

- **Lexicon namespace:** `app.hekate.*`. Don't add records under other namespaces in this repo.
- **AT-URI references in records:** always strings in the format `at://did:.../<collection>/<rkey>`.
- **State transitions:** spec'd in `docs/SPEC.md`. Indexer enforces; PDS records are unconstrained.
- **Commits:** Conventional Commits (`feat(server):`, `docs(adr):`, `fix(cli):` …) with DCO sign-off (`-s`).
- **No CLA**, only DCO. Don't add a CLA bot.
- **Generated code** under `server/internal/lexicon/` is committed (per ADR 0001).

## What NOT to do

- Don't proxy writes through the server. Clients write directly to their PDS; the server is read-only.
- Don't add a centralised registry of venues or instances. Federation is load-bearing.
- Don't introduce a search/cache layer (Elasticsearch, Redis) without an ADR. Postgres handles current scale.
- Don't promote the web client back onto the critical path without first revisiting the salt-mines verdict.
- Don't ship `app.hekate.event` or `app.hekate.rsvp` from a non-test PDS until ADR 0002 closes (M-1).
- Don't run a full server build casually — use `go vet ./...` for quick checks; build only when needed.

## When the user asks about scope

Default to: ship M0, then M1 + M2 + M2.5 (grant) in parallel; defer M3 until M1 has external readers. The kill date is 2026-10-29. If a request would push past that, raise it explicitly.
