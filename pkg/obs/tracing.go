package obs

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type TracingProvider struct {
	provider *sdktrace.TracerProvider
	config   Config
}

func newTracingProvider(ctx context.Context, config Config) (*TracingProvider, error) {
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

	if len(config.ResourceAttributes) > 0 {
		var customAttrs []attribute.KeyValue
		for key, value := range config.ResourceAttributes {
			customAttrs = append(customAttrs, attribute.String(key, value))
		}
		customRes, err := resource.New(ctx, resource.WithAttributes(customAttrs...))
		if err != nil {
			return nil, fmt.Errorf("failed to create custom resource: %w", err)
		}
		res, err = resource.Merge(res, customRes)
		if err != nil {
			return nil, fmt.Errorf("failed to merge resources: %w", err)
		}
	}

	var spanProcessor sdktrace.SpanProcessor

	if config.OTLPEndpoint != "" {
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(config.OTLPEndpoint),
			otlptracehttp.WithTimeout(config.OTLPTimeout),
		}

		if config.OTLPInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}

		exporter, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}

		spanProcessor = sdktrace.NewBatchSpanProcessor(exporter)
	} else {
		spanProcessor = sdktrace.NewSimpleSpanProcessor(noopExporter{})
	}

	sampler := sdktrace.TraceIDRatioBased(config.TracingSampleRatio)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(spanProcessor),
		sdktrace.WithSampler(sampler),
	)

	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracingProvider{
		provider: provider,
		config:   config,
	}, nil
}

func (tp *TracingProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return tp.provider.Tracer(name, opts...)
}

func (tp *TracingProvider) Shutdown(ctx context.Context) error {
	return tp.provider.Shutdown(ctx)
}

func (tp *TracingProvider) ForceFlush(ctx context.Context) error {
	return tp.provider.ForceFlush(ctx)
}

type noopExporter struct{}

func (noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

func StartSpan(ctx context.Context, tracer trace.Tracer, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, opts...)
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}
