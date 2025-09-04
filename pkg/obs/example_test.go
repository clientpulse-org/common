package obs_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/quiby-ai/common/pkg/obs"
	"go.opentelemetry.io/otel/attribute"
)

func ExampleInit() {
	ctx := context.Background()

	config := obs.Config{
		ServiceName:    "my-service",
		ServiceVersion: "1.0.0",
		Environment:    "production",
		LogLevel:       "info",
		MetricsEnabled: true,
		MetricsPort:    9090,
	}

	o, err := obs.Init(ctx, config)
	if err != nil {
		panic(err)
	}
	defer o.Shutdown(ctx)

	tracer := obs.Tracer("my-service")
	ctx, span := tracer.Start(ctx, "example-operation")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", "123"),
		attribute.Int("items.count", 42),
	)

	obs.Info(ctx, "Processing request", "user_id", "123", "items", 42)

	meter := obs.Meter("my-service")
	counter, _ := meter.Int64Counter("requests_total")
	counter.Add(ctx, 1)

	fmt.Println("Observability initialized successfully")
}

func ExampleObservability_MetricsProvider() {
	ctx := context.Background()
	config := obs.DefaultConfig()
	config.ServiceName = "metrics-example"

	o, err := obs.Init(ctx, config)
	if err != nil {
		panic(err)
	}
	defer o.Shutdown(ctx)

	handler := o.MetricsProvider().HTTPHandler()

	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Metrics server configured")
}

func ExampleConfig() {
	config := obs.DefaultConfig()

	config.ServiceName = "my-api"
	config.ServiceVersion = "2.1.0"
	config.Environment = "staging"
	config.OTLPEndpoint = "https://otlp.example.com"
	config.TracingSampleRatio = 0.1 // Sample 10% of traces
	config.LogLevel = "debug"

	config.ResourceAttributes = map[string]string{
		"region":  "us-west-2",
		"cluster": "staging-1",
	}

	if err := config.Validate(); err != nil {
		panic(err)
	}

	fmt.Printf("Service: %s, Version: %s, Environment: %s\n",
		config.ServiceName, config.ServiceVersion, config.Environment)
}

func ExampleTracer() {
	ctx := context.Background()
	config := obs.DefaultConfig()
	config.ServiceName = "tracing-example"

	o, err := obs.Init(ctx, config)
	if err != nil {
		panic(err)
	}
	defer o.Shutdown(ctx)

	tracer := obs.Tracer("user-service")

	ctx, span := tracer.Start(ctx, "process-user-request")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", "user123"),
		attribute.String("operation", "update_profile"),
	)

	ctx, childSpan := tracer.Start(ctx, "validate-input")
	defer childSpan.End()

	obs.Info(ctx, "Validating user input", "user_id", "user123")

	time.Sleep(10 * time.Millisecond)

	childSpan.SetAttributes(attribute.Bool("validation.success", true))

	fmt.Println("Tracing example completed")
}
