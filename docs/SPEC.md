# hekate Lexicon Specification

This document is the narrative companion to `lexicons/app/hekate/*.json`. The JSON lexicons are normative; this document explains the model, state machine, and validation rules they imply.

## Goals

1. **Venue-first.** Venues exist as discoverable, addressable records before any event is attached. A venue with no events is still useful (geographic discovery, capacity advertisement, recurring slot publication).
2. **PDS-native.** All records live in the user/venue PDS. The indexer is read-only; it derives queryable state but is never authoritative.
3. **Federation-friendly.** No central server, no allowlist. Any indexer can index any PDS; multiple indexers can co-exist.
4. **Slot marketplace + recurring host.** Venues publish discrete slots OR RRULE templates that expand into slots; organizers claim a slot by publishing an event.

## Record types

```
app.hekate.venue       owned by venue-DID            "this place exists"
app.hekate.recurrence  owned by venue-DID, refs venue "this venue recurs on this schedule"
app.hekate.slot        owned by venue-DID, refs venue "this window is bookable"
app.hekate.event       owned by organizer-DID, refs slot "I am claiming this slot"
app.hekate.rsvp        owned by attendee-DID, refs event "I am attending"
```

DID-ownership is the access-control primitive: the venue owner alone can publish slots and recurrences against their venue; only the slot authoring DID can cancel a slot; an event's organizer alone can edit its title/description.

## State machine

A slot has three lifecycle states:

```
            (created by venue)
                  │
                  ▼
              [ open ]
                  │
                  ├──── (event published referencing this slot, valid claim) ────► [ claimed ]
                  │                                                                     │
                  └──── (venue owner deletes/updates with status=cancelled) ──┐         │
                                                                              ▼         │
                                                                          [ cancelled ] ◄┘
                                                                                        ▲
                                              (venue cancels a previously-claimed slot)─┘
```

**Valid claim** requires:
- `slot.status == "open"` at claim time.
- Event's `slot` AT-URI resolves to a real slot record.
- Event's authoring DID ≠ slot's authoring DID (i.e. organizer is distinct from venue) — relaxed for self-hosted personal venues per indexer policy.
- For `bookingPolicy == "review"` venues, the indexer holds the slot in a *pending* indexer-side state; the venue owner accepts by updating the slot record's `claimedBy` to point at the chosen event AT-URI. (`claimedBy` is indexer-set for `open` policy; venue-set for `review` policy.)

**Collision resolution.** If two events race for the same slot (both reference an `open` slot before either is observed), the indexer accepts the one with the earliest `createdAt`. The loser remains a valid record in its PDS but is surfaced as an unfulfilled claim by the indexer. This is a derived, not an on-PDS, decision.

## Validation rules (indexer-enforced)

The lexicon JSON enforces shape; the indexer enforces semantic rules:

| Rule | Failure mode |
|---|---|
| `recurrence.venue` and the recurrence record share an authoring DID | record dropped |
| `slot.venue` and the slot record share an authoring DID | record dropped |
| `slot.recurrence` (if set) resolves to a recurrence whose `venue` matches `slot.venue` | record dropped |
| `slot.end > slot.start` | record dropped |
| `slot.claimedBy` set by a non-owner DID for an `open`-policy venue | claim ignored, indexer-derived `claimedBy` wins |
| `event.slot` resolves to an existing slot and `event.createdAt >= slot.createdAt` | claim ignored |
| `rsvp.event` resolves; later `rsvp` from same DID supersedes earlier | older RSVPs hidden in derived state |
| RRULE parses against RFC 5545 | recurrence dropped |

Records that fail validation are still in the authoring PDS — the indexer's rejection is authoritative only for query results from that indexer instance.

## Discovery model

Indexers expose three primary axes:
1. **Geographic** — bbox / radius queries on `venue.geo`.
2. **Temporal** — slot windows, with materialised slots from recurrences.
3. **Identity** — venues owned by a DID; events organised by a DID; RSVPs from a DID.

PostGIS handles axis 1 natively. Axis 2 and 3 are plain B-trees on `start`, `end`, `venue_did`, `organizer_did`.

## What this lexicon set does NOT model

- **Ticketing / payments.** Out of scope. Use `event.externalUrl` to link out.
- **Comments / chat.** Out of scope. Use existing ATProto patterns (e.g. `app.bsky.feed.post` with replies linking to event AT-URIs).
- **Per-event RSVP-list visibility.** ATProto records are inherently public; this lexicon does not pretend otherwise.
- **Roles within an event.** No host/co-host/speaker model; the organizer is the authoring DID. Speaker/agenda info goes in `event.description` or `externalUrl`.

## Coordination with Smoke Signal

[Smoke Signal](https://smokesignal.events/) is a pre-existing AT Protocol events application (Rust) with substantial production usage. hekate is **venue-first** where Smoke Signal is **event-first**. The two models can co-exist:

- A venue in hekate could host events authored as Smoke Signal records; a hekate indexer MAY accept Smoke Signal event records pointing at hekate slots.
- ADR 0002 records the formal coordination outcome with the Smoke Signal maintainers (M-1).

## Versioning

Lexicons use the `$type` URI as their version pin. Breaking changes get a new `$type` (e.g. `app.hekate.venue.v2`); additive changes are made in place. Indexers SHOULD support all known `$type` versions and may filter by the highest version present.

The repo root `lexicons/` directory is the canonical source; published copies (e.g. on `lexicons.hekate.app`) are mirrors. The published URL resolves the `$type`.
