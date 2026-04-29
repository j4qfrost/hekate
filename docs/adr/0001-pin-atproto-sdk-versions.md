# ADR 0001 — Pin AT Protocol SDK versions; track Lexicon SDK redesign

- Status: Accepted
- Date: 2026-04-29
- Deciders: hekate maintainers

## Context

Per the [AT Protocol Spring 2026 Roadmap](https://atproto.com/ko/blog/2026-spring-roadmap), the current Lexicon SDK is being replaced with a Buf-inspired codegen and schema-management system. Reference SDKs across language ecosystems (including indigo, the Go reference) will receive lexicon-codegen improvements.

This affects hekate directly:
- `server/internal/lexicon/` is generated from `lexicons/app/hekate/*.json`.
- Both indigo and `@atproto/lexicon` will at some point ship the new codegen alongside the old; mid-cycle drift between server and web could break wire compatibility.

A salt-mines panel evaluation (2026-04-29) flagged this as a sev-4 risk.

## Decision

1. **Pin SDK versions in lockfiles.** `go.mod` pins the indigo version explicitly (no `latest` tags); `web/pnpm-lock.yaml` pins `@atproto/api` and `@atproto/lexicon`.
2. **One SDK migration at a time.** When the Buf-redesign lands in either ecosystem, the migration is a discrete PR with: regenerated types, an updated test, and a CHANGELOG entry referencing this ADR.
3. **Codegen output committed.** Generated lexicon code under `server/internal/lexicon/` is committed (not built fresh in CI), so contributors can reproduce builds without the toolchain.
4. **CI gate.** A `make lex-validate` step parses every lexicon against the upstream meta-schema (vendored copy or fetched on demand) and fails the build if any record diverges. A version drift check (`make lex-versions`) prints the pinned SDK versions and roadmap status.

## Consequences

- Slightly slower to adopt new SDK features. Acceptable for v0.1; revisit at v0.2.
- Manual migration burden when the Buf redesign lands. Mitigated by step 2's discrete-PR rule.
- Pinning is a maintenance signal: the project is alive when the pin moves forward in CHANGELOG.

## References

- AT Protocol Roadmap (Spring 2026): https://atproto.com/ko/blog/2026-spring-roadmap
- Bluesky Protocol Check-in (Fall 2025): https://docs.bsky.app/blog/protocol-checkin-fall-2025
- indigo: https://github.com/bluesky-social/indigo
