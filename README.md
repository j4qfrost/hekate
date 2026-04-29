# hekate

> Venue-first event coordination on the AT Protocol.

hekate is a federated event-coordination protocol and reference implementation. It models **venues** as first-class records: a place exists in the system before any event is attached to it. Venues publish discrete time-slots and/or RFC-5545 RRULE-based recurrences that expand into slots; organizers claim a slot by publishing an event record. All state lives in user and venue PDSes — the indexer is read-only and replaceable.

## Status

- **M0 (lexicons)**: in progress, target 2026-05-29.
- M1 (server), M2 (CLI), M2.5 (grant), M3 (web), M4 (reference instance) — see [`docs/ROADMAP.md`](docs/ROADMAP.md).

## Why venue-first?

Most event apps make the *event* primary and the venue a free-text field. That works for one-off conferences but breaks for the long tail: pubs with a Tuesday quiz, libraries with a monthly book club, makerspaces with weekly classes, gyms with open-mat hours. These places want a calendar of slots they can offer; organizers want to find a place that fits.

hekate flips the model. A venue exists once and offers slots forever. An event is what happens to claim a slot.

## What's in this repo

```
lexicons/      AT Protocol record schemas (the durable artifact)
server/        Go indexer + read API (M1)
cli/           Go CLI — the v0.1 reference client (M2)
web/           SvelteKit 2 web client (M3, off the critical path)
deploy/        Docker Compose for one-command self-hosting
docs/          Spec, architecture, self-host guide, roadmap, ADRs
```

## 5-minute self-host

```bash
git clone https://github.com/<owner>/hekate.git
cd hekate/deploy
docker compose up -d
```

Server on `:8080`, web on `:5173`, Postgres+PostGIS managed automatically. See [`docs/SELFHOST.md`](docs/SELFHOST.md) for production hardening.

## Contributing

Apache-2.0, DCO sign-off (no CLA). See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the contributor ladder and good-first-issue map. Architecture decisions are documented in [`docs/adr/`](docs/adr/).

## Coordination with Smoke Signal

[Smoke Signal](https://smokesignal.events/) is an established AT Protocol events application. hekate is venue-first where Smoke Signal is event-first; the two models can co-exist or align. See [`docs/adr/0002-smoke-signal-relationship.md`](docs/adr/0002-smoke-signal-relationship.md) for the active coordination.

## License

Apache-2.0. See [`LICENSE`](LICENSE).
