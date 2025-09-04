package obs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracingProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config without OTLP endpoint",
			config: Config{
				ServiceName:        "test-service",
				ServiceVersion:     "1.0.0",
				Environment:        "test",
				TracingSampleRatio: 1.0,
			},
			wantErr: false,
		},
		{
			name: "valid config with custom resource attributes",
			config: Config{
				ServiceName:        "test-service",
				ServiceVersion:     "1.0.0",
				Environment:        "test",
				TracingSampleRatio: 0.5,
				ResourceAttributes: map[string]string{
					"custom.attribute": "value",
					"another.attr":     "another-value",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with invalid OTLP endpoint (should not error, just use noop)",
			config: Config{
				ServiceName:        "test-service",
				ServiceVersion:     "1.0.0",
				Environment:        "test",
				OTLPEndpoint:       "invalid-endpoint",
				OTLPTimeout:        5 * time.Second,
				TracingSampleRatio: 1.0,
			},
			wantErr: false, // Should not error, just create with noop behavior
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := newTracingProvider(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)

				tracer := provider.Tracer("test-tracer")
				assert.NotNil(t, tracer)

				ctx, span := tracer.Start(ctx, "test-span")
				assert.NotNil(t, span)
				span.End()

				err = provider.Shutdown(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestTracingProviderMethods(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		TracingSampleRatio: 1.0,
	}

	provider, err := newTracingProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer func() {
		err := provider.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	tracer := provider.Tracer("test-tracer")
	assert.NotNil(t, tracer)

	err = provider.ForceFlush(ctx)
	assert.NoError(t, err)
}

func TestTraceIDAndSpanID(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		TracingSampleRatio: 1.0,
	}

	provider, err := newTracingProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer func() {
		err := provider.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	tracer := provider.Tracer("test-tracer")

	assert.Empty(t, TraceID(ctx))
	assert.Empty(t, SpanID(ctx))

	spanCtx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	traceID := TraceID(spanCtx)
	spanID := SpanID(spanCtx)

	assert.NotEmpty(t, traceID)
	assert.NotEmpty(t, spanID)
	assert.Len(t, traceID, 32) // Trace ID should be 32 hex characters
	assert.Len(t, spanID, 16)  // Span ID should be 16 hex characters
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		TracingSampleRatio: 1.0,
	}

	provider, err := newTracingProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer func() {
		err := provider.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	tracer := provider.Tracer("test-tracer")

	spanCtx, span := StartSpan(ctx, tracer, "test-span")
	defer span.End()

	assert.NotNil(t, span)
	assert.NotEqual(t, ctx, spanCtx) // Context should be different with span

	retrievedSpan := SpanFromContext(spanCtx)
	assert.Equal(t, span, retrievedSpan)
}

func TestNoopExporter(t *testing.T) {
	ctx := context.Background()
	exporter := noopExporter{}

	err := exporter.ExportSpans(ctx, nil)
	assert.NoError(t, err)

	err = exporter.Shutdown(ctx)
	assert.NoError(t, err)
}
