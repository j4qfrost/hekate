# Contributing to hekate

Thanks for considering a contribution. hekate is Apache-2.0 with DCO sign-off (no CLA), so contributing is low-friction by design.

## TL;DR

- Find an issue tagged `good-first-issue`. Comment "I'll take this" so you don't double-up.
- Fork, branch, make your change, sign-off your commits with `git commit -s`.
- Open a PR. Fill in the template (it's short).
- CI runs; a maintainer reviews; we merge.

## Setup

You need:

- **Go 1.22+** for `server/` and `cli/`.
- **Node 22+ and pnpm 9+** for `web/`.
- **Docker** for the integration tests and the `make selfhost-smoke` target.

A one-command dev loop:

```bash
make dev
```

That starts Postgres+PostGIS in Docker, runs `hekate-server` with `air` for hot-reload, and runs the SvelteKit dev server. Ctrl-C stops everything cleanly.

## DCO sign-off

We use the [Developer Certificate of Origin](https://developercertificate.org/) instead of a CLA. Every commit must have a `Signed-off-by` line, which `git commit -s` adds automatically. No paperwork; just confirm you wrote (or have rights to) the code you're contributing.

## Issue ladder

Issues are labelled by friendliness:

| Label | What it means |
|---|---|
| `good-first-issue` | Self-contained, scoped, no surprise dependencies. Picked specifically for first-time contributors. Examples: add a filter to `/venues`; add a CLI flag; tighten a lexicon field. |
| `help-wanted` | Real work that we'd like a community contribution on. Some context required. Examples: add a new HTTP endpoint; build a small library. |
| `discussion` | A design question. Comment first; PRs only after agreement on the approach. |
| (unlabelled) | Maintainer-driven; PRs welcome but expect coordination. |

If a `good-first-issue` doesn't have a clear acceptance criterion in the description, ask for it in a comment. We'll fix the description, not your PR.

## What we want help with most

1. **Lexicon review.** If you've worked with AT Protocol lexicons elsewhere, a fresh pair of eyes on `lexicons/app/hekate/*.json` is high-value.
2. **First-issue ladder maintenance.** Spotting issues that *would* be a `good-first-issue` if the description were a bit clearer.
3. **Self-host smoke testing on platforms we don't run.** macOS, Windows+WSL, ARM64.
4. **Coordination with [Smoke Signal](https://smokesignal.events/).** See [`docs/adr/0002-smoke-signal-relationship.md`](docs/adr/0002-smoke-signal-relationship.md).

## Code style

- **Go:** standard `gofmt`. We do not use `goimports` ordering opinions; group stdlib, third-party, internal.
- **TypeScript:** Biome on default. Run `pnpm --filter web check`.
- **Lexicons:** 2-space JSON indent. Property order: `type`, then required scalars, then optional scalars, then arrays/objects, then `createdAt` last.
- **Commit messages:** [Conventional Commits](https://www.conventionalcommits.org/) — `type(scope): subject`. Examples: `feat(server): add /venues bbox filter`, `docs(spec): clarify slot-claim race resolution`, `fix(cli): handle 404 on get`.

## Tests

- `make test` runs everything: `go test ./...` in server and cli; `pnpm --filter web check && pnpm --filter web test`; lexicon validation.
- Integration tests live alongside their package and use `testcontainers-go` for a real Postgres.
- Web tests use Vitest; component tests prefer `@testing-library/svelte`.

PRs without tests for new behavior will be politely sent back to add them. Refactors and doc-only PRs are exempt.

## Architecture decisions

We track significant choices in [`docs/adr/`](docs/adr/). If your PR changes the protocol, the indexer's responsibilities, or a major dep, write an ADR alongside the code change. Use `docs/adr/0001-pin-atproto-sdk-versions.md` as a template.

## Coordination etiquette

- Open an issue *before* a PR for anything bigger than a one-file change. Saves rework if the answer is "we don't want this."
- Don't pile on issues someone else has commented "I'll take this" on. If they go quiet for a week, it's fair game; ping them in the comments first.
- Be kind in reviews. We are all part-timing this.
