package obs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggingProvider(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				LogLevel:       "info",
				LogPretty:      false,
				LogRedactText:  true,
				LogHashPII:     true,
			},
			wantErr: false,
		},
		{
			name: "debug log level",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				LogLevel:       "debug",
				LogPretty:      true,
				LogRedactText:  false,
				LogHashPII:     false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := newLoggingProvider(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.NotNil(t, provider.logger)
				assert.Equal(t, tt.config, provider.config)
			}
		})
	}
}

func TestLoggingProviderMethods(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		LogLevel:       "debug",
		LogPretty:      false,
		LogRedactText:  false,
		LogHashPII:     false,
	}

	provider, err := newLoggingProvider(config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	logger := provider.Logger()
	assert.NotNil(t, logger)

	tracingLogger := provider.WithTracing(ctx)
	assert.NotNil(t, tracingLogger)

	provider.Debug(ctx, "debug message", "key", "value")
	provider.Info(ctx, "info message", "key", "value")
	provider.Warn(ctx, "warn message", "key", "value")
	provider.Error(ctx, "error message", errors.New("test error"), "key", "value")
	provider.Event(ctx, "test_event", "ok", "key", "value")

	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestLoggingProviderWithTracing(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		LogLevel:           "debug",
		TracingSampleRatio: 1.0,
		MetricsPort:        9090,
	}

	obs, err := Init(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, obs)
	defer func() {
		err := obs.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	tracer := obs.Tracer("test-tracer")
	spanCtx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	tracingLogger := obs.logging.WithTracing(spanCtx)
	assert.NotNil(t, tracingLogger)

	obs.logging.Debug(spanCtx, "debug with tracing", "key", "value")
	obs.logging.Info(spanCtx, "info with tracing", "key", "value")
	obs.logging.Warn(spanCtx, "warn with tracing", "key", "value")
	obs.logging.Error(spanCtx, "error with tracing", errors.New("test error"), "key", "value")
	obs.logging.Event(spanCtx, "traced_event", "ok", "key", "value")
}

func TestGlobalLoggingFunctions(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		LogLevel:       "debug",
		MetricsPort:    9090,
	}

	globalMu.Lock()
	globalObs = nil
	globalMu.Unlock()

	Debug(ctx, "debug message", "key", "value")
	Info(ctx, "info message", "key", "value")
	Warn(ctx, "warn message", "key", "value")
	Error(ctx, "error message", errors.New("test error"), "key", "value")
	Event(ctx, "test_event", "ok", "key", "value")

	obs, err := Init(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, obs)
	defer func() {
		err := obs.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	Debug(ctx, "debug message with obs", "key", "value")
	Info(ctx, "info message with obs", "key", "value")
	Warn(ctx, "warn message with obs", "key", "value")
	Error(ctx, "error message with obs", errors.New("test error"), "key", "value")
	Event(ctx, "test_event_with_obs", "ok", "key", "value")
}

func TestLoggingWithTracingIntegration(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		LogLevel:           "debug",
		TracingSampleRatio: 1.0,
		MetricsPort:        9090,
	}

	globalMu.Lock()
	globalObs = nil
	globalMu.Unlock()

	obs, err := Init(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, obs)
	defer func() {
		err := obs.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	tracer := obs.Tracer("test-tracer")

	spanCtx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	traceID := TraceID(spanCtx)
	spanID := SpanID(spanCtx)

	assert.NotEmpty(t, traceID)
	assert.NotEmpty(t, spanID)

	Debug(spanCtx, "debug with global trace", "operation", "test")
	Info(spanCtx, "info with global trace", "operation", "test")
	Warn(spanCtx, "warn with global trace", "operation", "test")
	Error(spanCtx, "error with global trace", errors.New("test error"), "operation", "test")
	Event(spanCtx, "traced_global_event", "ok", "operation", "test")
}
