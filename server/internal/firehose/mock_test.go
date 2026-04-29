package firehose

import (
	"context"
	"testing"
	"time"
)

func TestMockEmitsThenStops(t *testing.T) {
	m := &Mock{Events: []Event{
		{DID: "did:plc:a", Collection: "app.hekate.venue", RKey: "1"},
		{DID: "did:plc:b", Collection: "app.hekate.slot", RKey: "2"},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	sink := make(chan Event, 2)
	if err := m.Run(ctx, sink); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	close(sink)

	got := 0
	for range sink {
		got++
	}
	if got != 2 {
		t.Errorf("expected 2 events emitted, got %d", got)
	}
}
