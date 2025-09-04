package obs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "unknown", config.ServiceName)
	assert.Equal(t, "dev", config.ServiceVersion)
	assert.Equal(t, "development", config.Environment)
	assert.Equal(t, "", config.OTLPEndpoint)
	assert.False(t, config.OTLPInsecure)
	assert.Equal(t, 30*time.Second, config.OTLPTimeout)
	assert.Equal(t, 1.0, config.TracingSampleRatio)
	assert.True(t, config.MetricsEnabled)
	assert.Equal(t, "/metrics", config.MetricsPath)
	assert.Equal(t, 9090, config.MetricsPort)
	assert.Equal(t, "info", config.LogLevel)
	assert.False(t, config.LogPretty)
	assert.True(t, config.LogRedactText)
	assert.True(t, config.LogHashPII)
	assert.NotNil(t, config.ResourceAttributes)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: nil,
		},
		{
			name: "valid config with custom values",
			config: Config{
				ServiceName:        "test-service",
				ServiceVersion:     "1.0.0",
				Environment:        "production",
				TracingSampleRatio: 0.5,
				MetricsPort:        8080,
			},
			wantErr: nil,
		},
		{
			name: "empty service name",
			config: Config{
				ServiceName:        "",
				TracingSampleRatio: 1.0,
				MetricsPort:        9090,
			},
			wantErr: ErrInvalidServiceName,
		},
		{
			name: "invalid sample ratio - negative",
			config: Config{
				ServiceName:        "test-service",
				TracingSampleRatio: -0.1,
				MetricsPort:        9090,
			},
			wantErr: ErrInvalidSampleRatio,
		},
		{
			name: "invalid sample ratio - greater than 1",
			config: Config{
				ServiceName:        "test-service",
				TracingSampleRatio: 1.5,
				MetricsPort:        9090,
			},
			wantErr: ErrInvalidSampleRatio,
		},
		{
			name: "invalid metrics port - zero",
			config: Config{
				ServiceName:        "test-service",
				TracingSampleRatio: 1.0,
				MetricsPort:        0,
			},
			wantErr: ErrInvalidMetricsPort,
		},
		{
			name: "invalid metrics port - negative",
			config: Config{
				ServiceName:        "test-service",
				TracingSampleRatio: 1.0,
				MetricsPort:        -1,
			},
			wantErr: ErrInvalidMetricsPort,
		},
		{
			name: "invalid metrics port - too high",
			config: Config{
				ServiceName:        "test-service",
				TracingSampleRatio: 1.0,
				MetricsPort:        65536,
			},
			wantErr: ErrInvalidMetricsPort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
