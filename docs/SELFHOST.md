# Self-host hekate in 5 minutes

This guide brings up the full hekate stack on a single machine using Docker Compose: Postgres+PostGIS, the Go indexer/API, and the SvelteKit web client. No external dependencies, no API keys.

## Prerequisites

- Docker 24+ and Docker Compose v2.
- ~500 MB of disk for the Postgres volume + container images.
- Outbound HTTPS access to your chosen ATProto relay (default `bsky.network`).

That is the entire prerequisite list. Node, Go, sqlc, and goose are NOT required for self-hosting — they are build-time tools, not runtime tools.

## One command

```bash
git clone https://github.com/<owner>/hekate.git
cd hekate/deploy
docker compose up -d
```

That is all. After ~30 seconds:

- `http://localhost:8080/healthz` returns `{"ok":true}` from the Go server.
- `http://localhost:5173` shows the SvelteKit web client.
- `docker compose logs -f hekate-server` shows the firehose subscription in action.

## Verify it works

The repo includes `make selfhost-smoke` which runs `docker compose up -d`, waits for `/healthz`, asserts the web root returns 200, and tears down. Use this in CI or to sanity-check a fresh deploy.

```bash
cd ..
make selfhost-smoke
```

Expected: `selfhost smoke OK`.

## Configuration (optional)

Every knob has a sensible default. Override via `deploy/.env`:

| Variable | Default | Meaning |
|---|---|---|
| `HEKATE_RELAY_URL` | `wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos` | ATProto firehose to subscribe to. Point at your own relay if you run one. |
| `HEKATE_LISTEN` | `:8080` | Server bind address inside the container. |
| `HEKATE_DATABASE_URL` | (auto, points at the postgis service) | Postgres URL. Override only if you bring your own database. |
| `HEKATE_RECURRENCE_HORIZON_DAYS` | `90` | How far ahead the recurrence expander materialises slots. |
| `WEB_PUBLIC_API_URL` | `http://localhost:8080` | What the web client uses to reach the server. Set to your public origin in production. |

## Production checklist

For real deployments, beyond `docker compose up`:

- [ ] Put a TLS terminator in front (Caddy, Traefik, nginx) — neither hekate-server nor the SvelteKit dev server speaks TLS directly.
- [ ] Use an external Postgres with backups enabled. The compose-managed Postgres is fine for small instances but has no built-in backup.
- [ ] Set `HEKATE_RELAY_URL` to a relay you trust to be available — `bsky.network` is fine but you may prefer a community relay for political reasons.
- [ ] Watch `docker compose logs hekate-server` for `firehose: connected` and `indexer: caught up`.
- [ ] Pin the image tag (don't run `latest`). Each release ships an immutable digest; see `CHANGELOG.md`.
- [ ] Set up a `/healthz` ping in your monitor of choice.

## Troubleshooting

**`/healthz` returns 503.** The indexer is not connected to the firehose. Check `docker compose logs hekate-server` — common causes are a blocked outbound WebSocket (corp firewalls) or a wrong `HEKATE_RELAY_URL`.

**Web shows "no venues nearby".** That is correct for a fresh deployment — the indexer has not yet seen any `app.hekate.venue` records. Publish one with the CLI (`hekate venue create …`) and watch it appear.

**Postgres container exits.** Check `docker compose logs postgis`. Likely culprits: insufficient shared-memory (set `shm_size: 512mb` in the compose file if needed) or a stale volume from a prior install (drop with `docker compose down -v`).

**Slow firehose backlog.** Increase `HEKATE_INDEXER_BATCH_SIZE` (default 100) or run multiple `hekate-server` replicas pointed at the same Postgres — they coordinate via row-level locks.

## Uninstall

```bash
cd deploy
docker compose down -v
```

`-v` drops the Postgres volume. Without it, the data persists for the next bring-up.
