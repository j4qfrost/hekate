package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	r := NewReader(srv.URL)
	if err := r.Healthz(context.Background()); err != nil {
		t.Fatalf("Healthz: %v", err)
	}
}

func TestHealthzReportsNotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false}`))
	}))
	defer srv.Close()

	r := NewReader(srv.URL)
	if err := r.Healthz(context.Background()); err == nil {
		t.Error("expected error when server reports ok=false")
	}
}

func TestHealthzNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	r := NewReader(srv.URL)
	if err := r.Healthz(context.Background()); err == nil {
		t.Error("expected error for 503")
	}
}
