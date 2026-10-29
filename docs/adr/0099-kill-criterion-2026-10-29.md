# ADR 0099: Kill criterion executed — scope reduced to lexicons-only (2026-10-29)

- **Status:** Accepted
- **Date:** 2026-10-29
- **Deciders:** j4qfrost (precommitted 2026-04-29)

## Context

From `docs/ROADMAP.md` M1 kill criterion paragraph (verbatim):

> **Kill criterion (precommitted):** if M1 has not shipped reads-from-external-PDS by **2026-10-29**, scope down to lexicons-only and pause. This is not negotiable mid-flight.

This criterion was established on 2026-04-29 as part of the salt-mines portfolio evaluation, which classified hekate as a credibility play rather than a cash project. The rationale was to hedge against the user's documented apps-stall pattern (`feedback_finish_apps`): if the server never shipped, the lexicons alone would still be a useful protocol contribution. The precommitment was designed specifically to take future-self wishful thinking out of the loop.

## Evidence evaluated on 2026-10-29

1. **ROADMAP M1 checkboxes:** All eight checkboxes under the M1 section remain `[ ]` unchecked, including "Reads at least one external PDS's records end-to-end."

2. **Git activity since 2026-04-29:** Significant M1 groundwork was laid:
   - `feat(server): wire indigo firehose; pin SDKs` (2026-04-29, commit 79d3316)
   - `feat(server): implement PgWriter for indexer` (2026-05-18, commit 7bde849)
   - `feat(server): wire firehose-indexer pipeline into hekate-server` (2026-05-20, commit e4120a3)
   - `test(server): add external PDS round-trip integration test for M1 kill-criterion` (2026-07-01, commit dd47e8d)

3. **Integration test status:** The test `TestExternalPDSRoundTrip` in `server/internal/indexer/indexer_integration_test.go` is well-implemented and targets bsky.social/bsky.network. However it skips unconditionally when `HEKATE_TEST_HANDLE` or `HEKATE_TEST_APP_PASSWORD` are absent. The Woodpecker CI pipeline (`.woodpecker.yml`) runs only `go vet ./... && go build ./...` — it never supplies these credentials. No CI run has ever executed the test with real credentials. The test has never been reported as passing.

4. **Firehose implementation state:** `grep -rn 'ErrNotImplemented' server/internal/firehose/` returns no results — the firehose is not a stub. Indigo (`github.com/bluesky-social/indigo`) is present in `server/go.mod`. The server builds. But a building server with an untested integration path does not satisfy "ships reads from external PDS."

5. **Open PRs:** One open PR (#7, `m2.5-grant-draft`) — the M2.5 grant application. Zero open PRs related to M1 completion or the integration test credentials.

6. **Hours-from-shipping test:** No PR exists with passing CI that adds the firehose credentials to CI or marks the round-trip test as having passed against a real PDS. The "7-day extension" exception does not apply.

## Decision

The kill criterion is triggered. Scope is reduced to lexicons-only as of 2026-10-29.

The `docs/ROADMAP.md` M1 section is marked `[KILLED 2026-10-29]`. Milestones M2 (CLI), M2.5 (grant), M3 (web), M4 (reference instance), and M5 (mobile) are marked `[OUT OF SCOPE — paused per kill criterion]`. M0 (lexicons) remains the active, maintained milestone.

## Consequences

- `server/`, `cli/`, `web/`, `deploy/`, and `companion/` remain in-tree at their last commit as historical context. They are not deleted.
- The maintained surface area is `lexicons/app/hekate/` and `docs/SPEC.md`.
- `README.md` is updated to a lexicons-only framing.
- The project is not abandoned — it is paused at a well-defined scope. If a future session ships the external-PDS round-trip with a real PDS (e.g. by adding credentials to CI and landing a green run), this ADR can be superseded by an "M1 resumed" ADR and the milestones re-opened.
- The AT Protocol community can use the `app.hekate.*` lexicons with any compliant indexer regardless of this scope decision.
