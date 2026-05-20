# ADR 0003 — AT Protocol grant submission outcome

- Status: **Draft** (M2.5; application not yet submitted)
- Date opened: 2026-05-18
- Application target deadline: 2026-06-29 (per `docs/ROADMAP.md` M2.5)
- Deciders: hekate maintainers; AT Protocol grants program (decision)

## Context

The salt-mines panel evaluation (2026-04-29; eval `51be2c20-5558-41c1-b60c-90b843edb15d`) flagged AT Protocol grants as the **only realistic monetisation path** for hekate. The verdict is "credibility play, not a cash project" — but submitting a grant application has value independent of the outcome because it forces a written, external-facing scope artifact.

Per the M2.5 roadmap entry: regardless of approval, the submission itself is a milestone deliverable. ADR 0003 records both the submission and the response.

## Pre-submission status (filled at draft time)

| Item | Status |
|---|---|
| M0 lexicons frozen | In progress; gated by ADR 0002 (Smoke Signal coordination) |
| `docs/SPEC.md` published | Yes |
| `docs/ARCHITECTURE.md` published | Yes |
| `docs/ROADMAP.md` published | Yes |
| Reference indexer (Go) | Indexer writer landed 2026-05-18; firehose→writer pipe still pending |
| Reference client (Go CLI) | Stubs only |
| Public repo | https://github.com/j4qfrost/hekate (Apache-2.0) |

## Decision

*To be filled after submission and response.*

Two decision points are recorded here:

### 1. Submission decision (decided locally, no external dependency)

- [ ] Application drafted
- [ ] Application submitted via the active AT Protocol grants channel
- [ ] Submission link / reference recorded

### 2. Grant outcome (decided by the grants program)

- [ ] Approved — record the funding scope and any conditions; update `docs/ROADMAP.md` to reflect funded work
- [ ] Declined — record the stated reason and decide whether to revise+resubmit; **continue M1/M2 on existing trajectory regardless**
- [ ] No response within 90 days — treat as declined for planning purposes

## Application scope (draft outline)

The application narrative should cover:

1. **What hekate adds to the AT Protocol ecosystem.** Venue-first event coordination is novel relative to Smoke Signal (event-first) — see ADR 0002. The `venue`/`slot`/`recurrence` lexicons fill an addressable gap.
2. **What's already shipped.** M0 lexicons, indexer architecture, ADRs 0001/0002/0004/0005, companion stream engine. Cite the public repo.
3. **What grant funding would unlock.** Specific, dated work that does NOT extend past M1's 2026-10-29 kill date. Candidate items:
   - Buf-redesign lexicon SDK migration once upstream lands (ADR 0001 follow-up)
   - Reference deployment at a non-author-owned PDS to demonstrate federation
   - Smoke Signal coordination follow-through (option B or C from ADR 0002, if landed)
4. **What this is NOT.** A request to fund the user's salary, infrastructure, or app-layer work — funding is for the *protocol contributions*. The salt-mines panel's "credibility play" verdict frames this honestly.

## Consequences

- **Submitting forces M0 to be ship-ready.** Pulls ADR 0002 resolution and M0 freeze onto the critical path of M2.5. Reasonable.
- **Approval would not change M1's kill date** — the firehose-reading-an-external-PDS line is a technical milestone, not a funding one.
- **Declined ≠ shutdown.** Per the salt-mines verdict, hekate's value is the artifacts (lexicons + indexer + spec), not the funding.

## References

- `docs/ROADMAP.md` M2.5 — application deadline 2026-06-29
- `docs/adr/0002-smoke-signal-relationship.md` — must close before submission
- Salt-mines eval `51be2c20-5558-41c1-b60c-90b843edb15d` — credibility-play verdict
- AT Protocol grants program — channel TBD (research before drafting application)
