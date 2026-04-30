# Observability

hekate ships an opt-in LGTM observability stack — Loki, Grafana, Tempo, Mimir — wired up via OpenTelemetry. See [ADR 0005](adr/0005-lgtm-observability-stack.md) for the rationale; this doc is the operator's guide.

## TL;DR

```bash
make obs-up                                      # start the stack
HEKATE_OTLP_ENDPOINT=localhost:4317 \
HEKATE_OTLP_INSECURE=true \
  go run ./server/cmd/hekate-server               # point server at it
open http://localhost:3001/d/hekate-overview      # Grafana UI
make obs-down                                     # tear it down
```

Anonymous Grafana login is on by default (everyone is org Admin). For real deployments, replace the all-in-one container with separate Loki / Tempo / Mimir / Grafana instances.

## What you get

The bring-up uses the [`grafana/otel-lgtm`](https://github.com/grafana/docker-otel-lgtm) all-in-one image. Ports exposed on localhost:

| Service | URL | Purpose |
|---|---|---|
| Grafana | http://localhost:3001 | Dashboards + Explore (anonymous Admin) |
| OTLP/gRPC | localhost:4317 | What hekate-server exports to |
| OTLP/HTTP | localhost:4318 | Alternative for non-gRPC clients |
| Loki | http://localhost:3100 | Log storage (LogQL) |
| Tempo | http://localhost:3200 | Trace storage (TraceQL) |
| Mimir | http://localhost:9009 | Metric storage (PromQL) |

Pre-provisioned dashboard at `/d/hekate-overview` gives:

**Venue users row**
- Distinct authoring DIDs in the last hour
- Venues created / updated / deleted (rate)
- Venue records indexed after validation

**Events processing row**
- Slot lifecycle (open → claimed → cancelled) and event-claim attempts
- RSVP counts by status (going / maybe / declined)
- Recurrence slots materialised (daily expander tick)

**Indexer health row**
- Firehose decode errors by reason (read_car / get_record / …)
- Indexer validation errors by collection + reason
- Commit-handle latency p50 / p95 / p99
- HTTP read-API requests/sec by route + status

## Wiring the server

Set two env vars and the OTel SDK lights up:

```bash
HEKATE_OTLP_ENDPOINT=localhost:4317   # required to enable export
HEKATE_OTLP_INSECURE=true             # localhost dev → no TLS
HEKATE_SERVICE_VERSION=v0.1.0         # optional; recorded as service.version
HEKATE_ENVIRONMENT=dev                # optional; recorded as deployment.environment
```

If `HEKATE_OTLP_ENDPOINT` is empty, the SDK is initialised with no-op exporters — instruments still register, calls into them are cheap, but nothing is exported. This means `make selfhost-smoke` runs unchanged whether or not the observability stack is up.

## Metric reference

All metric names come from `server/internal/telemetry/telemetry.go`. The labels are the only contract; metric names are stable for v0.1.

| Metric | Type | Labels | Emitted by |
|---|---|---|---|
| `hekate_firehose_events_total` | counter | `collection`, `action` | firehose consumer per relevant op |
| `hekate_firehose_decode_errors_total` | counter | `reason` (`read_car`, `get_record`), `collection?` | firehose, on CAR / record decode failure |
| `hekate_firehose_handle_duration_seconds` | histogram | (none) | firehose, per relevant commit |
| `hekate_indexer_records_indexed_total` | counter | `collection`, `action` | indexer (M1) per typed-table write |
| `hekate_indexer_validation_errors_total` | counter | `collection`, `reason` | indexer when a record fails semantic validation |
| `hekate_recurrence_slots_materialized_total` | counter | (none) | recurrence expander on daily tick |
| `http_server_request_duration_seconds` | histogram | `http_route`, `http_response_status_code`, … | otelhttp middleware |

Spans:

| Span | Attributes | Emitted by |
|---|---|---|
| `firehose.handle_commit` | `hekate.did`, `atproto.commit.seq`, `atproto.commit.ops` | firehose consumer per relevant commit |
| HTTP request spans | standard otelhttp set + route override | api router |

## Production deployments

The all-in-one image is great for laptops and snowman; it is **not** sized for production. For real deployments:

1. Run separate Loki, Tempo, Mimir, and Grafana instances (Helm charts exist; or roll your own compose).
2. Put a [Grafana Alloy](https://grafana.com/docs/alloy/latest/) gateway in front. Alloy receives OTLP from hekate-server and fans out to each backend.
3. Point `HEKATE_OTLP_ENDPOINT` at the Alloy gateway.
4. Copy `deploy/observability/grafana/dashboards/hekate-overview.json` to your Grafana provisioning path.

The OTLP wire format is the only commitment hekate makes; pick whatever backends you want behind it.

## Troubleshooting

**Grafana shows "no data".** First check that hekate-server is actually pointed at the OTLP endpoint (look for `telemetry.Init` succeeding in server logs). Then check that you launched the server *with* the env var set — the SDK only configures exporters at boot.

**Distinct DID stat shows 0 even though events are flowing.** The panel uses `count(count by (hekate_did) ...)` which depends on the `hekate.did` resource attribute being attached to metric series. The current implementation attaches it as a span attribute only; for distinct-user panels you want to either propagate it to a metric label (raises cardinality concerns) or query traces via TraceQL. The dashboard ships with the metric query as a placeholder; tighten it once cardinality bounds are agreed.

**Container health flaps.** The `grafana/otel-lgtm` image takes ~30 s to fully come up. Healthcheck retries 12× at 10 s; if it's still failing after 2 min, check `docker logs hekate-lgtm`.

**Where do I look for traces?** Grafana → Explore → datasource Tempo → query e.g. `{name="firehose.handle_commit"}` or filter by attribute `hekate.did`.

## What's NOT instrumented yet

- **Logs to Loki.** Server still emits `log/slog` JSON to stdout; structured-log shipping is deferred until M1 indexer logs are worth aggregating.
- **CLI traces.** Outbound traces from `hekate` CLI commands would help debug round-trip issues but aren't worth the dependency surface yet.
- **Web client RUM.** M3 web is off the critical path; RUM lands when the web client does.
