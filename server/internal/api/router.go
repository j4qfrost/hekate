// Package api wires the read-only HTTP API. Routes are mounted with the
// stdlib net/http enhanced mux (Go 1.22+). All endpoints are anonymous and
// cacheable; writes happen client-to-PDS, not through this server.
package api

import (
	"encoding/json"
	"net/http"
)

// NewRouter returns the read API mux with M0 endpoints stubbed. Real
// query handlers land with M1.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /venues", handleVenues)
	mux.HandleFunc("GET /slots", handleSlots)
	mux.HandleFunc("GET /events", handleEvents)
	return mux
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
