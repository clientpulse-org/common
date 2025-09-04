package obs

import (
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type MetricsProvider struct {
	provider *sdkmetric.MeterProvider
	registry *prometheus.Registry
	exporter *promexporter.Exporter
	config   Config
}

func newMetricsProvider(ctx context.Context, config Config) (*MetricsProvider, error) {
	if !config.MetricsEnabled {
		return &MetricsProvider{config: config}, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	registry := prometheus.NewRegistry()

	exporter, err := promexporter.New(
		promexporter.WithRegisterer(registry),
		promexporter.WithoutUnits(),
		promexporter.WithoutScopeInfo(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
	)

	otel.SetMeterProvider(provider)

	return &MetricsProvider{
		provider: provider,
		registry: registry,
		exporter: exporter,
		config:   config,
	}, nil
}

func (mp *MetricsProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	if mp.provider == nil {
		return otel.Meter(name, opts...)
	}
	return mp.provider.Meter(name, opts...)
}

func (mp *MetricsProvider) HTTPHandler() http.Handler {
	if mp.registry == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(mp.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

func (mp *MetricsProvider) Registry() *prometheus.Registry {
	return mp.registry
}

func (mp *MetricsProvider) Shutdown(ctx context.Context) error {
	if mp.provider == nil {
		return nil
	}
	return mp.provider.Shutdown(ctx)
}

func (mp *MetricsProvider) ForceFlush(ctx context.Context) error {
	if mp.provider == nil {
		return nil
	}
	return mp.provider.ForceFlush(ctx)
}

func (mp *MetricsProvider) Counter(name, description, unit string) (metric.Int64Counter, error) {
	meter := mp.Meter("github.com/quiby-ai/common/obs")
	return meter.Int64Counter(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
}

func (mp *MetricsProvider) Histogram(name, description, unit string) (metric.Float64Histogram, error) {
	meter := mp.Meter("github.com/quiby-ai/common/obs")
	return meter.Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
}

func (mp *MetricsProvider) Gauge(name, description, unit string) (metric.Float64ObservableGauge, error) {
	meter := mp.Meter("github.com/quiby-ai/common/obs")
	return meter.Float64ObservableGauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
}

func (mp *MetricsProvider) UpDownCounter(name, description, unit string) (metric.Int64UpDownCounter, error) {
	meter := mp.Meter("github.com/quiby-ai/common/obs")
	return meter.Int64UpDownCounter(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
}
