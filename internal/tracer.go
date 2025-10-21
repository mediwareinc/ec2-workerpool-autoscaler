package internal

import (
	"context"
	"net/http"
)

// Tracer is a cloud-agnostic interface for distributed tracing.
// It provides methods for capturing segments, adding metadata and annotations,
// and instrumenting HTTP clients.
type Tracer interface {
	// Capture wraps a function in a tracing segment/span with the given name.
	// The wrapped function receives a context that should be used for any
	// child operations.
	Capture(ctx context.Context, name string, fn func(context.Context) error) error

	// AddMetadata adds metadata to the current segment/span in the context.
	// Metadata is searchable but not indexed.
	AddMetadata(ctx context.Context, key string, value interface{})

	// AddAnnotation adds an annotation to the current segment/span in the context.
	// Annotations are indexed and searchable.
	AddAnnotation(ctx context.Context, key string, value interface{})

	// Client returns an HTTP client instrumented with tracing.
	// If client is nil, a default HTTP client will be used.
	Client(client *http.Client) *http.Client

	// Configure initializes the tracer with the given configuration.
	Configure(config TracerConfig) error

	// Begin starts a new root segment with the given name.
	// Returns a context with the segment and a function to close the segment.
	Begin(ctx context.Context, name string) (context.Context, func(error))
}

// TracerConfig holds configuration for the tracer.
type TracerConfig struct {
	// ServiceVersion is the version of the service being traced.
	ServiceVersion string

	// Enabled indicates whether tracing is enabled.
	Enabled bool
}
