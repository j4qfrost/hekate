# ADR 0002 — Relationship with Smoke Signal

- Status: **Pending** (M-1; awaiting coordination)
- Date opened: 2026-04-29
- Deciders: hekate maintainers, Smoke Signal maintainers (Nick Gerakines et al.)

## Context

[Smoke Signal](https://smokesignal.events/) is an established AT Protocol events and RSVP application (Rust, by Nick Gerakines, ~18-month head start as of 2026-04-29). It is event-first: users create events and RSVP. It does not currently model venues as first-class records; venue information is denormalised into the event.

hekate proposes a *venue-first* model: venues, slots, and recurrences are first-class records, and an event is a slot-claim. The two models are complementary in principle but overlap on the event/RSVP primitives.

A salt-mines panel evaluation (2026-04-29) flagged Smoke Signal's head start and potential expansion into venue primitives as sev-4 risks. The panel recommended formal coordination *before* publishing hekate lexicons.

## Options under consideration

**A. Coexist — divergent specs.** hekate ships `app.hekate.event` and `app.hekate.rsvp`; Smoke Signal keeps theirs. Indexers from each project ignore the other's records. Loses interop; fastest to ship.

**B. hekate-as-venue-only.** hekate ships only `venue`, `slot`, and `recurrence`; events and RSVPs are delegated to Smoke Signal's lexicon (`events.smokesignal.calendar.event` or equivalent). hekate indexers index Smoke Signal events whose `slot` (or equivalent venue reference) points at a hekate slot. Maximum interop; depends on Smoke Signal accepting an extension field for slot reference.

**C. Joint standard.** Co-author a shared `community.lexicon.event.*` (or similar) namespace with Smoke Signal that both projects implement. Slowest; requires meaningful agreement.

## Decision

*To be filled in after the M-1 coordination thread completes.* The coordination ask:
1. Open a thread on https://discourse.smokesignal.events/ summarising hekate's venue-first model and proposing options A/B/C.
2. Mirror the thread on the SmokeSignal-Events GitHub org as a discussion if the discourse host prefers.
3. Allow ~2 weeks for response. If no response, default to **A (coexist)** with a public note that the coordination was attempted.

The recorded decision MUST be written into this ADR before M0 ships (i.e. before any `app.hekate.event` or `app.hekate.rsvp` record is published from a non-test PDS).

## Consequences

- This ADR is the gate that opens or closes the event/rsvp lexicons. Until it is decided, those two lexicon files are subject to change.
- The venue/slot/recurrence lexicons are NOT blocked by this ADR — they are uniquely hekate's contribution regardless of outcome.

## References

- Smoke Signal: https://smokesignal.events/
- Smoke Signal docs: https://docs.smokesignal.events/
- Smoke Signal discourse: https://discourse.smokesignal.events/
- Smoke Signal GitHub org: https://github.com/SmokeSignal-Events
- Tech talk (2024): https://atprotocol.dev/tech-talk-smoke-signal-events/
