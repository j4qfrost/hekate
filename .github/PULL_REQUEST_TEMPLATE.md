## What

<!-- Brief: what does this change do? -->

## Why

<!-- Brief: what was wrong, missing, or worth doing? Link the issue if there is one. -->

## Checklist

- [ ] Commits are signed-off (`git commit -s`) — DCO, no CLA.
- [ ] Tests added or updated for new behaviour (refactors and doc-only PRs exempt).
- [ ] If this changes a lexicon (`lexicons/app/hekate/*.json`), `docs/SPEC.md` is updated to match.
- [ ] If this changes the protocol surface or a major dep, an ADR is added under `docs/adr/`.
- [ ] `make test` passes locally (or CI is green).

## ATProto SDK pin (per ADR 0001)

- [ ] No SDK version was bumped, OR
- [ ] SDK bump is intentional and the CHANGELOG entry references ADR 0001.
