// Package telemetry initialises OpenTelemetry traces and metrics for
// hekate-server and exposes hekate-specific metric instruments.
//
// When HEKATE_OTLP_ENDPOINT is unset, the SDK is configured with no-op
// exporters: instruments record into in-memory accumulators that are
// discarded, and tracing spans are no-op. This means callers can use
// telemetry.Tracer() and telemetry.Meter() unconditionally without an
// "is telemetry enabled" branch — the cost in the unset case is a
// handful of nanoseconds per call.
//
// See docs/adr/0005-lgtm-observability-stack.md for the why.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// ServiceName is the service.name resource attribute applied to every
// emitted span and metric series.
const ServiceName = "hekate-server"

// Config controls OTel initialisation. Construct via FromEnv() in normal
// use; tests can build it directly.
type Config struct {
	OTLPEndpoint string // host:port for OTLP/gRPC; empty disables network exports
	Insecure     bool   // skip TLS to the collector (true for local dev)
	ServiceVersion string
	Environment  string // e.g. "dev", "prod"; recorded as deployment.environment
}

// FromEnv reads HEKATE_OTLP_*  env vars and returns a Config.
func FromEnv() Config {
	return Config{
		OTLPEndpoint:   os.Getenv("HEKATE_OTLP_ENDPOINT"),
		Insecure:       os.Getenv("HEKATE_OTLP_INSECURE") != "false",
		ServiceVersion: envOr("HEKATE_SERVICE_VERSION", "dev"),
		Environment:    envOr("HEKATE_ENVIRONMENT", "dev"),
	}
}

// Provider holds the SDK objects so the caller can call Shutdown on exit.
type Provider struct {
	tp        *sdktrace.TracerProvider
	mp        *sdkmetric.MeterProvider
	shutdowns []func(context.Context) error
}

// Shutdown flushes any pending exports and closes the gRPC connection.
// Safe to call with a non-running Provider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var errs []error
	for _, fn := range p.shutdowns {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Init wires the global OTel providers. Returns a Provider whose Shutdown
// the caller MUST defer. If cfg.OTLPEndpoint is empty, returns a Provider
// with no-op exporters (Shutdown is a no-op too).
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	res, err := buildResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}

	p := &Provider{}

	// Trace provider.
	tpOpts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if cfg.OTLPEndpoint != "" {
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		tex, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("telemetry: trace exporter: %w", err)
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(tex,
			sdktrace.WithBatchTimeout(5*time.Second)))
		p.shutdowns = append(p.shutdowns, tex.Shutdown)
	}
	p.tp = sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(p.tp)
	p.shutdowns = append(p.shutdowns, p.tp.Shutdown)

	// Metric provider.
	mpOpts := []sdkmetric.Option{sdkmetric.WithResource(res)}
	if cfg.OTLPEndpoint != "" {
		opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint)}
		if cfg.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		mex, err := otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("telemetry: metric exporter: %w", err)
		}
		mpOpts = append(mpOpts, sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(mex, sdkmetric.WithInterval(15*time.Second))))
		p.shutdowns = append(p.shutdowns, mex.Shutdown)
	}
	p.mp = sdkmetric.NewMeterProvider(mpOpts...)
	otel.SetMeterProvider(p.mp)
	p.shutdowns = append(p.shutdowns, p.mp.Shutdown)

	if err := registerInstruments(p.mp); err != nil {
		return nil, fmt.Errorf("telemetry: register instruments: %w", err)
	}
	return p, nil
}

func buildResource(cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		attribute.String("deployment.environment", cfg.Environment),
	}
	host, _ := os.Hostname()
	if host != "" {
		attrs = append(attrs, attribute.String("host.name", host))
	}
	return resource.Merge(resource.Default(), resource.NewWithAttributes("", attrs...))
}

// Tracer returns a tracer scoped to the named instrumentation library.
// Safe to call before Init() — returns a no-op tracer in that case.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// Meter returns a meter scoped to the named instrumentation library.
// Safe to call before Init() — returns a no-op meter.
func Meter(name string) metric.Meter {
	return otel.Meter(name)
}

// Hekate-specific instruments. Registered once by Init; safe to use
// concurrently. Code that emits to these does so via the package-level
// vars below rather than re-registering each call.

var (
	registerOnce sync.Once

	FirehoseEvents          metric.Int64Counter
	FirehoseDecodeErrors    metric.Int64Counter
	FirehoseHandleDuration  metric.Float64Histogram
	IndexerRecords          metric.Int64Counter
	IndexerValidationErrors metric.Int64Counter
	RecurrenceSlotsMaterialized metric.Int64Counter
)

func registerInstruments(mp *sdkmetric.MeterProvider) error {
	var regErr error
	registerOnce.Do(func() {
		m := mp.Meter("github.com/j4qfrost/hekate/server")

		var err error
		if FirehoseEvents, err = m.Int64Counter(
			"hekate_firehose_events_total",
			metric.WithDescription("Total app.hekate.* records observed on the firehose, labelled by collection and action."),
			metric.WithUnit("{record}"),
		); err != nil {
			regErr = err
			return
		}
		if FirehoseDecodeErrors, err = m.Int64Counter(
			"hekate_firehose_decode_errors_total",
			metric.WithDescription("Firehose decode failures, labelled by reason."),
			metric.WithUnit("{error}"),
		); err != nil {
			regErr = err
			return
		}
		if FirehoseHandleDuration, err = m.Float64Histogram(
			"hekate_firehose_handle_duration_seconds",
			metric.WithDescription("Wall-clock time spent handling a single relevant commit, end-to-end."),
			metric.WithUnit("s"),
		); err != nil {
			regErr = err
			return
		}
		if IndexerRecords, err = m.Int64Counter(
			"hekate_indexer_records_indexed_total",
			metric.WithDescription("Records materialised into typed tables by the indexer, labelled by collection and action."),
			metric.WithUnit("{record}"),
		); err != nil {
			regErr = err
			return
		}
		if IndexerValidationErrors, err = m.Int64Counter(
			"hekate_indexer_validation_errors_total",
			metric.WithDescription("Records rejected by indexer-side semantic validation, labelled by collection and reason."),
			metric.WithUnit("{record}"),
		); err != nil {
			regErr = err
			return
		}
		if RecurrenceSlotsMaterialized, err = m.Int64Counter(
			"hekate_recurrence_slots_materialized_total",
			metric.WithDescription("Slots materialised by the recurrence expander."),
			metric.WithUnit("{slot}"),
		); err != nil {
			regErr = err
			return
		}
	})
	return regErr
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
