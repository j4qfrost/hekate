// Package client wraps two concerns:
//   - REST reads against the hekate-server (anonymous, no auth).
//   - PDS writes against an authenticated AT Protocol session (M2; OAuth via
//     the upstream @atproto/oauth-client-go equivalent or app-password).
//
// At v0.1 only the read client is functional; M2 ships the writer.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Reader is a thin REST client for hekate-server.
type Reader struct {
	BaseURL string
	HTTP    *http.Client
}

// NewReader returns a Reader with a 10s default timeout.
func NewReader(baseURL string) *Reader {
	return &Reader{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Healthz returns nil if the server reports {"ok":true}.
func (r *Reader) Healthz(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.BaseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := r.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz: status %d", resp.StatusCode)
	}
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("healthz: decode: %w", err)
	}
	if !body.OK {
		return errors.New("healthz: server reports not ok")
	}
	return nil
}
