# ADR 0005 — LGTM observability stack via OpenTelemetry

- Status: Accepted
- Date: 2026-04-29
- Deciders: hekate maintainers

## Context

The salt-mines panel evaluation (2026-04-29) flagged that the user's apps-stall pattern usually shows up *after* the first surge of work, when the operator can no longer eyeball the system. Without observability, hekate-server is opaque the moment it leaves the laptop: was the firehose connected? are records being decoded? are venues actually being indexed? are slot-claim races visible?

We want answers to those questions to be one click away in a dashboard. Specifically the operator needs to see, at a glance:

- Firehose connection status + lag from relay sequence head.
- Per-collection record throughput: `app.hekate.{venue,slot,recurrence,event,rsvp}` × `{create,update,delete}`.
- Indexer error rate by reason (CAR decode, CBOR decode, validation, write).
- Venue-user signal: distinct authoring DIDs over time, geographic concentration of new venues.
- Event-processing signal: slot claims observed per minute; collisions resolved.
- HTTP read API: requests per second, p50/p99 latency, error rate by route.

## Decision

**Adopt the LGTM stack (Loki + Grafana + Tempo + Mimir) via OpenTelemetry as the standard observability surface for hekate.**

Concretely:

1. **Server instrumentation** lives in `server/internal/telemetry/`. It initialises the OTel SDK (traces + metrics) with an OTLP/gRPC exporter targeting `HEKATE_OTLP_ENDPOINT`. When the env var is unset, exporters are no-ops and the server runs unmodified. This keeps `make selfhost-smoke` green without LGTM running.
2. **`net/http` middleware** uses `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` to record `http_server_*` metrics and request spans for `/healthz`, `/venues`, `/slots`, `/events`.
3. **Hekate-specific metrics** are registered in `telemetry`:
   - `hekate_firehose_events_total{collection,action}` — counter
   - `hekate_firehose_decode_errors_total{reason}` — counter
   - `hekate_firehose_handle_duration_seconds` — histogram
   - `hekate_indexer_records_indexed_total{collection,action}` — counter (used when M1 lands)
   - `hekate_indexer_validation_errors_total{collection,reason}` — counter
   - `hekate_recurrence_slots_materialized_total` — counter
4. **Dev / single-host LGTM** ships as `deploy/observability/docker-compose.yml`, using the `grafana/otel-lgtm:0.8.1` all-in-one image. This is for laptops, snowman, and self-host smoke tests — **not** for production. It is **not** part of `deploy/docker-compose.yml`; the 5-minute self-host promise excludes the observability stack.
5. **Production deployments** are encouraged to run separate Loki / Tempo / Mimir / Grafana instances and point `HEKATE_OTLP_ENDPOINT` at an Alloy (or otel-collector) gateway. This ADR does not prescribe the production topology; the OTLP contract is the only commitment.
6. **Pre-provisioned Grafana dashboard** ships at `deploy/observability/grafana/dashboards/hekate-overview.json` and is auto-loaded via Grafana's provisioning sidecar. Two rows: "Venue users" and "Events processing".

## Why LGTM over alternatives

- **Prometheus + Jaeger + plain log files.** Workable, but three separate query languages, three separate UIs, and no log-trace-metric correlation. LGTM's single Grafana pane gives free correlation via TraceQL → LogQL → PromQL.
- **ELK / Datadog / Honeycomb / NewRelic.** SaaS or heavy. Conflicts with the OSS adoption pillar — contributors should be able to reproduce the dashboards locally without a paid account.
- **Just Prometheus.** Tempting for v0.1 simplicity, but a federated event indexer where slot-claim races matter benefits enormously from distributed traces.
- **OpenTelemetry as the wire format** is non-negotiable: it's the vendor-neutral hedge if we ever swap backends.

## Consequences

- Server gains OTel SDK + exporter dependencies (~10 packages, all under `go.opentelemetry.io/...`). ADR 0001 pinned-versions table updated alongside this ADR.
- `go test ./...` runtime increases trivially (~ms) — exporters are silently no-op'd in tests.
- Self-host story stays single-command for the production compose; observability is opt-in via `make obs-up`.
- Documentation gains a new entrypoint (`docs/OBSERVABILITY.md`).
- One additional Make target group (`obs-up`, `obs-down`, `obs-status`).

## Out of scope

- **Logs:** stdlib `log/slog` JSON output is fine for v0.1; structured-log-to-Loki shipping is deferred until M1 actually has logs worth aggregating.
- **Real user monitoring (RUM)** in the SvelteKit web client. The web client is M3, off the critical path.
- **Synthetic uptime checks.** Belongs at a higher layer (UptimeRobot, Cloudflare Healthchecks); not hekate's job.

## References

- OpenTelemetry Go SDK: https://opentelemetry.io/docs/languages/go/
- Grafana otel-lgtm image: https://github.com/grafana/docker-otel-lgtm
- LGTM stack overview: https://grafana.com/oss/lgtm/
