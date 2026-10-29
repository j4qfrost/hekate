# hekate

> Venue-first event lexicons on the AT Protocol.

hekate is a venue-first event lexicon set on the AT Protocol. The Go server, CLI, and SvelteKit web client originally planned around it were paused on 2026-10-29 per the project's precommitted kill criterion ([docs/ROADMAP.md](docs/ROADMAP.md)). The lexicons remain the durable artifact and are usable by any AT Protocol indexer.

The lexicons model **venues** as first-class records: a place exists in the system before any event is attached to it. Venues publish discrete time-slots and/or RFC-5545 RRULE-based recurrences that expand into slots; organizers claim a slot by publishing an event record. All state lives in user and venue PDSes.

## Why venue-first?

Most event apps make the *event* primary and the venue a free-text field. That works for one-off conferences but breaks for the long tail: pubs with a Tuesday quiz, libraries with a monthly book club, makerspaces with weekly classes, gyms with open-mat hours. These places want a calendar of slots they can offer; organizers want to find a place that fits.

hekate flips the model. A venue exists once and offers slots forever. An event is what happens to claim a slot.

## Lexicons

The lexicon schemas are in [`lexicons/app/hekate/`](lexicons/app/hekate/) and documented in [`docs/SPEC.md`](docs/SPEC.md):

- `app.hekate.venue` — a physical or virtual place that offers time-slots
- `app.hekate.slot` — a single bookable time window offered by a venue
- `app.hekate.recurrence` — an RRULE-based recurring availability pattern
- `app.hekate.event` — an organizer's claim of a slot *(subject to change pending ADR 0002)*
- `app.hekate.rsvp` — an attendee's RSVP to an event *(subject to change pending ADR 0002)*

## What's in this repo

```
lexicons/      AT Protocol record schemas (the maintained artifact)
server/        Go indexer + read API (paused at last commit)
cli/           Go CLI (paused at last commit)
web/           SvelteKit 2 web client (paused at last commit)
deploy/        Docker Compose (paused at last commit)
docs/          Spec, architecture, roadmap, ADRs
```

The paused components remain in-tree at their last commit as historical context but are no longer built or shipped.

## Contributing

Apache-2.0, DCO sign-off (no CLA). See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the contributor ladder and good-first-issue map. Architecture decisions are documented in [`docs/adr/`](docs/adr/).

## Coordination with Smoke Signal

[Smoke Signal](https://smokesignal.events/) is an established AT Protocol events application. hekate is venue-first where Smoke Signal is event-first; the two models can co-exist or align. See [`docs/adr/0002-smoke-signal-relationship.md`](docs/adr/0002-smoke-signal-relationship.md) for the active coordination.

## License

Apache-2.0. See [`LICENSE`](LICENSE).
