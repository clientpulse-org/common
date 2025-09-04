package obs

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsProvider(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "metrics enabled",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				MetricsEnabled: true,
			},
			wantErr: false,
		},
		{
			name: "metrics disabled",
			config: Config{
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Environment:    "test",
				MetricsEnabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := newMetricsProvider(ctx, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)

				if tt.config.MetricsEnabled {
					assert.NotNil(t, provider.registry)
					assert.NotNil(t, provider.exporter)
					assert.NotNil(t, provider.provider)
				} else {
					assert.Nil(t, provider.registry)
					assert.Nil(t, provider.exporter)
					assert.Nil(t, provider.provider)
				}

				meter := provider.Meter("test-meter")
				assert.NotNil(t, meter)

				err = provider.Shutdown(ctx)
				assert.NoError(t, err)
			}
		})
	}
}

func TestMetricsProviderMethods(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		MetricsEnabled: true,
	}

	provider, err := newMetricsProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer func() {
		err := provider.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	meter := provider.Meter("test-meter")
	assert.NotNil(t, meter)

	handler := provider.HTTPHandler()
	assert.NotNil(t, handler)
	assert.Implements(t, (*http.Handler)(nil), handler)

	registry := provider.Registry()
	assert.NotNil(t, registry)

	err = provider.ForceFlush(ctx)
	assert.NoError(t, err)
}

func TestMetricsProviderDisabled(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		MetricsEnabled: false,
	}

	provider, err := newMetricsProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	meter := provider.Meter("test-meter")
	assert.NotNil(t, meter)

	handler := provider.HTTPHandler()
	assert.NotNil(t, handler)

	registry := provider.Registry()
	assert.Nil(t, registry)

	err = provider.ForceFlush(ctx)
	assert.NoError(t, err)

	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestMetricsInstruments(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		MetricsEnabled: true,
	}

	provider, err := newMetricsProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer func() {
		err := provider.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	counter, err := provider.Counter("test_counter", "A test counter", "1")
	assert.NoError(t, err)
	assert.NotNil(t, counter)

	counter.Add(ctx, 1)

	histogram, err := provider.Histogram("test_histogram", "A test histogram", "ms")
	assert.NoError(t, err)
	assert.NotNil(t, histogram)

	histogram.Record(ctx, 100.0)

	gauge, err := provider.Gauge("test_gauge", "A test gauge", "bytes")
	assert.NoError(t, err)
	assert.NotNil(t, gauge)

	upDownCounter, err := provider.UpDownCounter("test_updown", "A test up/down counter", "1")
	assert.NoError(t, err)
	assert.NotNil(t, upDownCounter)

	upDownCounter.Add(ctx, 5)
	upDownCounter.Add(ctx, -2)
}

func TestMetricsInstrumentsDisabled(t *testing.T) {
	ctx := context.Background()
	config := Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		MetricsEnabled: false,
	}

	provider, err := newMetricsProvider(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, provider)

	counter, err := provider.Counter("test_counter", "A test counter", "1")
	assert.NoError(t, err)
	assert.NotNil(t, counter)

	histogram, err := provider.Histogram("test_histogram", "A test histogram", "ms")
	assert.NoError(t, err)
	assert.NotNil(t, histogram)

	gauge, err := provider.Gauge("test_gauge", "A test gauge", "bytes")
	assert.NoError(t, err)
	assert.NotNil(t, gauge)

	upDownCounter, err := provider.UpDownCounter("test_updown", "A test up/down counter", "1")
	assert.NoError(t, err)
	assert.NotNil(t, upDownCounter)
}
