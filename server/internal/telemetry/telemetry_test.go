package telemetry

import (
	"context"
	"testing"
)

func TestInitNoopWhenEndpointEmpty(t *testing.T) {
	p, err := Init(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer p.Shutdown(context.Background())

	// Instruments must be non-nil even with no exporter — instrumented code
	// dereferences them unconditionally.
	if FirehoseEvents == nil || FirehoseDecodeErrors == nil || FirehoseHandleDuration == nil {
		t.Error("instruments not registered for no-op provider")
	}
	if IndexerRecords == nil || IndexerValidationErrors == nil || RecurrenceSlotsMaterialized == nil {
		t.Error("indexer/recurrence instruments not registered")
	}
}

func TestEmitDoesNotPanicWithoutExporter(t *testing.T) {
	if _, err := Init(context.Background(), Config{}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Defensive: emitting before/after Shutdown must not panic.
	FirehoseEvents.Add(context.Background(), 1)
	FirehoseDecodeErrors.Add(context.Background(), 1)
	FirehoseHandleDuration.Record(context.Background(), 0.012)
	IndexerRecords.Add(context.Background(), 1)
	IndexerValidationErrors.Add(context.Background(), 1)
	RecurrenceSlotsMaterialized.Add(context.Background(), 13)
}

func TestTracerAndMeterReturnUsableImpls(t *testing.T) {
	tr := Tracer("test")
	if tr == nil {
		t.Error("Tracer returned nil")
	}
	m := Meter("test")
	if m == nil {
		t.Error("Meter returned nil")
	}
	_, span := tr.Start(context.Background(), "smoke")
	span.End()
}
