package internal

import (
	"context"
	"net/http"

	"github.com/aws/aws-xray-sdk-go/xray"
)

// XRayTracer is an implementation of the Tracer interface using AWS X-Ray.
type XRayTracer struct{}

// NewXRayTracer creates a new XRayTracer instance.
func NewXRayTracer() *XRayTracer {
	return &XRayTracer{}
}

// Capture wraps a function in an X-Ray segment.
func (t *XRayTracer) Capture(ctx context.Context, name string, fn func(context.Context) error) error {
	return xray.Capture(ctx, name, fn)
}

// AddMetadata adds metadata to the current X-Ray segment.
func (t *XRayTracer) AddMetadata(ctx context.Context, key string, value interface{}) {
	xray.AddMetadata(ctx, key, value)
}

// AddAnnotation adds an annotation to the current X-Ray segment.
func (t *XRayTracer) AddAnnotation(ctx context.Context, key string, value interface{}) {
	xray.AddAnnotation(ctx, key, value)
}

// Client returns an HTTP client instrumented with X-Ray.
func (t *XRayTracer) Client(client *http.Client) *http.Client {
	return xray.Client(client)
}

// Configure initializes X-Ray with the given configuration.
func (t *XRayTracer) Configure(config TracerConfig) error {
	return xray.Configure(xray.Config{
		ServiceVersion: config.ServiceVersion,
	})
}

// Begin starts a new X-Ray root segment.
func (t *XRayTracer) Begin(ctx context.Context, name string) (context.Context, func(error)) {
	newCtx, segment := xray.BeginSegment(ctx, name)
	return newCtx, func(err error) {
		segment.Close(err)
	}
}
