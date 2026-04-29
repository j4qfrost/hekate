# Postgres logical-replication settings

The companion stream-processing engine (RisingWave; see ADR 0004) consumes
`events_decoded` via Postgres CDC. CDC requires logical replication, which is
**off by default** in the `postgis/postgis:16-3.4` image used by both the
production and dev compose files.

## Required settings

```
wal_level             = logical
max_replication_slots = 10
max_wal_senders       = 10
```

`wal_level` cannot be changed without a Postgres restart, so an init script
under `/docker-entrypoint-initdb.d/` is **not** sufficient.

## How to apply (compose)

Override the container command so Postgres starts with the flags:

```yaml
services:
  postgis:
    command: >
      postgres
      -c wal_level=logical
      -c max_replication_slots=10
      -c max_wal_senders=10
```

The companion stack at `deploy/companion/docker-compose.yml` already does this.

## Production-compose integration

The root `deploy/docker-compose.yml` is intentionally **unchanged** while the
companion stack lives separately. When RisingWave joins the production compose
(M1+), the `command:` override above must be added to the `postgis` service
there too. Until then, do not add it — the smoke target's footprint stays as
documented in `docs/SELFHOST.md`.
