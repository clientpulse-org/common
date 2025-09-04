package obs

import (
	"time"
)

type Config struct {
	ServiceName        string            `env:"SERVICE_NAME" envDefault:"unknown"`
	ServiceVersion     string            `env:"SERVICE_VERSION" envDefault:"dev"`
	Environment        string            `env:"ENV" envDefault:"development"`
	OTLPEndpoint       string            `env:"OTLP_ENDPOINT" envDefault:""`
	OTLPInsecure       bool              `env:"OTLP_INSECURE" envDefault:"false"`
	OTLPTimeout        time.Duration     `env:"OTLP_TIMEOUT" envDefault:"30s"`
	TracingSampleRatio float64           `env:"TRACING_SAMPLE_RATIO" envDefault:"1.0"`
	MetricsEnabled     bool              `env:"METRICS_ENABLED" envDefault:"true"`
	MetricsPath        string            `env:"METRICS_PATH" envDefault:"/metrics"`
	MetricsPort        int               `env:"METRICS_PORT" envDefault:"9090"`
	LogLevel           string            `env:"LOG_LEVEL" envDefault:"info"`
	LogPretty          bool              `env:"LOG_PRETTY" envDefault:"false"`
	LogRedactText      bool              `env:"LOG_REDACT_TEXT" envDefault:"true"`
	LogHashPII         bool              `env:"LOG_HASH_PII" envDefault:"true"`
	ResourceAttributes map[string]string `env:"RESOURCE_ATTRIBUTES"`
}

func DefaultConfig() Config {
	return Config{
		ServiceName:        "unknown",
		ServiceVersion:     "dev",
		Environment:        "development",
		OTLPEndpoint:       "",
		OTLPInsecure:       false,
		OTLPTimeout:        30 * time.Second,
		TracingSampleRatio: 1.0,
		MetricsEnabled:     true,
		MetricsPath:        "/metrics",
		MetricsPort:        9090,
		LogLevel:           "info",
		LogPretty:          false,
		LogRedactText:      true,
		LogHashPII:         true,
		ResourceAttributes: make(map[string]string),
	}
}

func (c Config) Validate() error {
	if c.ServiceName == "" {
		return ErrInvalidServiceName
	}
	if c.TracingSampleRatio < 0 || c.TracingSampleRatio > 1 {
		return ErrInvalidSampleRatio
	}
	if c.MetricsPort <= 0 || c.MetricsPort > 65535 {
		return ErrInvalidMetricsPort
	}
	return nil
}
