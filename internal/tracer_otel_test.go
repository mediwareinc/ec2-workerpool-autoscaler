package internal

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestOtelTracer_NewOtelTracer(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	if tracer == nil {
		t.Fatal("Expected tracer to be created")
	}
	if tracer.serviceName != "test-service" {
		t.Errorf("Expected service name to be 'test-service', got %s", tracer.serviceName)
	}
}

func TestOtelTracer_Configure_Disabled(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	err := tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        false,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if tracer.tracer == nil {
		t.Error("Expected tracer to be initialized")
	}
}

func TestOtelTracer_Configure_Enabled(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	err := tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if tracer.tracer == nil {
		t.Error("Expected tracer to be initialized")
	}
	if tracer.tracerProvider == nil {
		t.Error("Expected tracer provider to be initialized")
	}

	// Clean up
	_ = tracer.Shutdown(context.Background())
}

func TestOtelTracer_Capture(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	called := false
	err := tracer.Capture(ctx, "test-operation", func(ctx context.Context) error {
		called = true
		// Verify that a span exists in context
		span := trace.SpanFromContext(ctx)
		if !span.IsRecording() {
			t.Error("Expected span to be recording")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected function to be called")
	}
}

func TestOtelTracer_Capture_WithError(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	expectedErr := errors.New("test error")
	err := tracer.Capture(ctx, "test-operation", func(ctx context.Context) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestOtelTracer_AddMetadata(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	ctx, span := tracer.tracer.Start(context.Background(), "test")
	defer span.End()

	// Should not panic
	tracer.AddMetadata(ctx, "key", "value")
	tracer.AddMetadata(ctx, "number", 42)
	tracer.AddMetadata(ctx, "float", 3.14)
	tracer.AddMetadata(ctx, "bool", true)
}

func TestOtelTracer_AddAnnotation(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	ctx, span := tracer.tracer.Start(context.Background(), "test")
	defer span.End()

	// Should not panic
	tracer.AddAnnotation(ctx, "key", "value")
	tracer.AddAnnotation(ctx, "number", 42)
}

func TestOtelTracer_Client(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	// Test with nil client
	client := tracer.Client(nil)
	if client == nil {
		t.Error("Expected client to be returned")
	}
	if client.Transport == nil {
		t.Error("Expected transport to be set")
	}

	// Test with existing client
	existingClient := &http.Client{}
	client = tracer.Client(existingClient)
	if client == nil {
		t.Error("Expected client to be returned")
	}
	if client.Transport == nil {
		t.Error("Expected transport to be set")
	}
}

func TestOtelTracer_Begin(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	newCtx, closeFn := tracer.Begin(ctx, "test-segment")

	if newCtx == nil {
		t.Error("Expected context to be returned")
	}
	if closeFn == nil {
		t.Error("Expected close function to be returned")
	}

	span := trace.SpanFromContext(newCtx)
	if !span.IsRecording() {
		t.Error("Expected span to be recording")
	}

	// Close the segment
	closeFn(nil)
}

func TestOtelTracer_Begin_WithError(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	ctx := context.Background()
	newCtx, closeFn := tracer.Begin(ctx, "test-segment")

	if newCtx == nil {
		t.Error("Expected context to be returned")
	}

	// Close with error - should not panic
	expectedErr := errors.New("test error")
	closeFn(expectedErr)
}

func TestOtelTracer_Shutdown(t *testing.T) {
	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})

	err := tracer.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Expected no error on shutdown, got %v", err)
	}

	// Test shutdown when tracer provider is nil
	tracer2 := NewOtelTracer("test-service")
	err = tracer2.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Expected no error on shutdown with nil provider, got %v", err)
	}
}

func TestConvertToAttribute(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "key", "value"},
		{"int", "key", 42},
		{"int64", "key", int64(42)},
		{"float64", "key", 3.14},
		{"bool", "key", true},
		{"string slice", "key", []string{"a", "b"}},
		{"int slice", "key", []int{1, 2}},
		{"int64 slice", "key", []int64{1, 2}},
		{"float64 slice", "key", []float64{1.1, 2.2}},
		{"bool slice", "key", []bool{true, false}},
		{"other type", "key", struct{ Name string }{Name: "test"}},
		{"nil", "key", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			attr := convertToAttribute(tt.key, tt.value)
			if attr.Key != "key" {
				t.Errorf("Expected key to be 'key', got %s", attr.Key)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"int", 42, "42"},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.value)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestOtelTracer_GlobalTracerProvider(t *testing.T) {
	// Save the current global tracer provider
	originalProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(originalProvider)

	tracer := NewOtelTracer("test-service")
	_ = tracer.Configure(TracerConfig{
		ServiceVersion: "1.0.0",
		Enabled:        true,
	})
	defer tracer.Shutdown(context.Background())

	// Verify global tracer provider was set
	globalProvider := otel.GetTracerProvider()
	if globalProvider == nil {
		t.Error("Expected global tracer provider to be set")
	}
}
