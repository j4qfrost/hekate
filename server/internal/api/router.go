// Package api wires the read-only HTTP API. Routes are mounted with the
// stdlib net/http enhanced mux (Go 1.22+). All endpoints are anonymous and
// cacheable; writes happen client-to-PDS, not through this server.
//
// Every route is wrapped in otelhttp middleware, which emits
// http_server_request_duration / http_server_active_requests metrics and
// a request span when telemetry is enabled (HEKATE_OTLP_ENDPOINT). When
// telemetry is disabled, the middleware is a thin pass-through.
package api

import (
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NewRouter returns the read API mux with M0 endpoints stubbed. Real
// query handlers land with M1.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", instrument("healthz", handleHealthz))
	mux.Handle("GET /venues", instrument("venues", handleVenues))
	mux.Handle("GET /slots", instrument("slots", handleSlots))
	mux.Handle("GET /events", instrument("events", handleEvents))
	return mux
}

// instrument wraps an HTTP handler in otelhttp with the route name set so
// the resulting span / metric series is keyed by the logical operation
// rather than the URL path (which can carry high-cardinality query strings).
func instrument(operation string, h http.HandlerFunc) http.Handler {
	return otelhttp.NewHandler(h, operation,
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents))
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleVenues(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"error":  "not_implemented",
		"detail": "M1 — venue queries not yet wired",
	})
}

func handleSlots(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"error":  "not_implemented",
		"detail": "M1 — slot queries not yet wired",
	})
}

func handleEvents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]any{
		"error":  "not_implemented",
		"detail": "M1 — event queries not yet wired",
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
