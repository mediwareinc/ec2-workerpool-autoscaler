package internal

import (
	"context"
	"net/http"
)

// NoOpTracer is a no-op implementation of the Tracer interface for testing.
type NoOpTracer struct{}

// NewNoOpTracer creates a new NoOpTracer instance.
func NewNoOpTracer() *NoOpTracer {
	return &NoOpTracer{}
}

// Capture executes the function without any tracing.
func (t *NoOpTracer) Capture(ctx context.Context, name string, fn func(context.Context) error) error {
	return fn(ctx)
}

// AddMetadata does nothing.
func (t *NoOpTracer) AddMetadata(ctx context.Context, key string, value interface{}) {}

// AddAnnotation does nothing.
func (t *NoOpTracer) AddAnnotation(ctx context.Context, key string, value interface{}) {}

// Client returns the provided HTTP client or a default one.
func (t *NoOpTracer) Client(client *http.Client) *http.Client {
	if client == nil {
		return http.DefaultClient
	}
	return client
}

// Configure does nothing and returns nil.
func (t *NoOpTracer) Configure(config TracerConfig) error {
	return nil
}

// Begin returns the provided context and a no-op close function.
func (t *NoOpTracer) Begin(ctx context.Context, name string) (context.Context, func(error)) {
	return ctx, func(error) {}
}
