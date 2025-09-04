package obs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: Config{
				ServiceName:        "test-service",
				ServiceVersion:     "1.0.0",
				Environment:        "test",
				TracingSampleRatio: 0.5,
				MetricsEnabled:     true,
				MetricsPort:        8080,
				LogLevel:           "debug",
			},
			wantErr: false,
		},
		{
			name: "invalid config - empty service name",
			config: Config{
				ServiceName:        "",
				TracingSampleRatio: 1.0,
				MetricsPort:        9090,
			},
			wantErr: true,
		},
		{
			name: "invalid config - bad sample ratio",
			config: Config{
				ServiceName:        "test-service",
				TracingSampleRatio: 2.0,
				MetricsPort:        9090,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalMu.Lock()
			globalObs = nil
			globalMu.Unlock()

			obs, err := Init(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, obs)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, obs)
				assert.True(t, obs.IsInitialized())
				assert.Equal(t, tt.config.ServiceName, obs.Config().ServiceName)

				globalObs := Global()
				assert.Equal(t, obs, globalObs)

				err = obs.Shutdown(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestMustInit(t *testing.T) {
	ctx := context.Background()

	globalMu.Lock()
	globalObs = nil
	globalMu.Unlock()

	config := DefaultConfig()
	config.ServiceName = "test-service"

	obs := MustInit(ctx, config)
	assert.NotNil(t, obs)
	assert.True(t, obs.IsInitialized())

	err := obs.Shutdown(ctx)
	assert.NoError(t, err)

	invalidConfig := Config{
		ServiceName: "", // Invalid
	}

	assert.Panics(t, func() {
		MustInit(ctx, invalidConfig)
	})
}

func TestObservabilityMethods(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.ServiceName = "test-service"

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
	assert.NotNil(t, tracer)

	meter := obs.Meter("test-meter")
	assert.NotNil(t, meter)

	logger := obs.Logger()
	assert.NotNil(t, logger)

	assert.NotNil(t, obs.TracingProvider())
	assert.NotNil(t, obs.MetricsProvider())
	assert.NotNil(t, obs.LoggingProvider())

	assert.Equal(t, config, obs.Config())
}

func TestGlobalFunctions(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.ServiceName = "test-service"

	globalMu.Lock()
	globalObs = nil
	globalMu.Unlock()

	tracer := Tracer("test-tracer")
	assert.NotNil(t, tracer) // Should return noop tracer

	meter := Meter("test-meter")
	assert.NotNil(t, meter) // Should return noop meter

	obs, err := Init(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, obs)
	defer func() {
		err := obs.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	tracer = Tracer("test-tracer")
	assert.NotNil(t, tracer)

	meter = Meter("test-meter")
	assert.NotNil(t, meter)

	err = Shutdown(ctx)
	assert.NoError(t, err)
}

func TestShutdown(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.ServiceName = "test-service"

	globalMu.Lock()
	globalObs = nil
	globalMu.Unlock()

	err := Shutdown(ctx)
	assert.ErrorIs(t, err, ErrNotInitialized)

	obs, err := Init(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, obs)

	err = obs.Shutdown(ctx)
	assert.NoError(t, err)

	obs, err = Init(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, obs)

	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	<-timeoutCtx.Done()

	err = obs.Shutdown(timeoutCtx)
	assert.NoError(t, err)
}

func TestConcurrentInit(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.ServiceName = "test-service"

	globalMu.Lock()
	globalObs = nil
	globalMu.Unlock()

	const numGoroutines = 10
	results := make(chan *Observability, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			obs, err := Init(ctx, config)
			if err != nil {
				errors <- err
			} else {
				results <- obs
			}
		}()
	}

	var obs *Observability
	var initErrors []error

	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			if obs == nil {
				obs = result
			}
			assert.Same(t, obs, result)
		case err := <-errors:
			initErrors = append(initErrors, err)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}

	assert.NotNil(t, obs)
	for _, err := range initErrors {
		t.Logf("Init error: %v", err)
	}

	if obs != nil {
		err := obs.Shutdown(ctx)
		assert.NoError(t, err)
	}
}
