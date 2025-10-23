package internal

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// OtelTracer is an implementation of the Tracer interface using OpenTelemetry.
type OtelTracer struct {
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
	serviceName    string
}

// NewOtelTracer creates a new OtelTracer instance with the given service name.
func NewOtelTracer(serviceName string) *OtelTracer {
	return &OtelTracer{
		serviceName: serviceName,
	}
}

// Configure initializes OpenTelemetry with the given configuration.
func (t *OtelTracer) Configure(config TracerConfig) error {
	if !config.Enabled {
		// If tracing is disabled, use a no-op tracer provider
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		t.tracer = otel.Tracer(t.serviceName)
		return nil
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(t.serviceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
	if err != nil {
		return err
	}

	// Create tracer provider with a batch span processor
	// Note: Exporter should be configured separately based on your backend
	// (e.g., OTLP, Jaeger, Zipkin, etc.)
	t.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		// Add span processors here when you configure an exporter
		// For example:
		// sdktrace.WithBatcher(otlptrace.New(ctx, exporter)),
	)

	// Set as global tracer provider
	otel.SetTracerProvider(t.tracerProvider)

	// Get a tracer for this service
	t.tracer = otel.Tracer(t.serviceName)

	return nil
}

// Capture wraps a function in an OpenTelemetry span.
func (t *OtelTracer) Capture(ctx context.Context, name string, fn func(context.Context) error) error {
	ctx, span := t.tracer.Start(ctx, name)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
	}

	return err
}

// AddMetadata adds an attribute to the current span in the context.
// In OpenTelemetry, both metadata and annotations are represented as attributes.
func (t *OtelTracer) AddMetadata(ctx context.Context, key string, value interface{}) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(convertToAttribute(key, value))
	}
}

// AddAnnotation adds an attribute to the current span in the context.
// In OpenTelemetry, annotations are represented as attributes with higher importance.
func (t *OtelTracer) AddAnnotation(ctx context.Context, key string, value interface{}) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(convertToAttribute(key, value))
	}
}

// Client returns an HTTP client instrumented with OpenTelemetry.
func (t *OtelTracer) Client(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}

	// Wrap the transport with OpenTelemetry instrumentation
	client.Transport = otelhttp.NewTransport(client.Transport)
	return client
}

// Begin starts a new root span with the given name.
// Returns a context with the span and a function to close the span.
func (t *OtelTracer) Begin(ctx context.Context, name string) (context.Context, func(error)) {
	newCtx, span := t.tracer.Start(ctx, name)

	return newCtx, func(err error) {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}
}

// Shutdown gracefully shuts down the tracer provider.
// This should be called when the application is shutting down.
func (t *OtelTracer) Shutdown(ctx context.Context) error {
	if t.tracerProvider != nil {
		return t.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// convertToAttribute converts a value to an OpenTelemetry attribute.
func convertToAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	case []int:
		return attribute.IntSlice(key, v)
	case []int64:
		return attribute.Int64Slice(key, v)
	case []float64:
		return attribute.Float64Slice(key, v)
	case []bool:
		return attribute.BoolSlice(key, v)
	default:
		// For other types, convert to string
		return attribute.String(key, toString(v))
	}
}

// toString converts a value to string representation.
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
