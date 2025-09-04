# pkg/obs - Unified Observability Package

A comprehensive observability package that provides OpenTelemetry tracing, Prometheus metrics, and structured JSON logging in a single, easy-to-use interface.

## Features

- **OpenTelemetry Tracing**: Distributed tracing with OTLP HTTP export
- **Prometheus Metrics**: Metrics collection and HTTP endpoint exposure
- **Structured Logging**: JSON logging with PII redaction and trace correlation
- **Unified Initialization**: Single `Init()` call to set up all observability components
- **Graceful Shutdown**: Proper cleanup of all resources
- **Environment-based Configuration**: Configure via environment variables

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/quiby-ai/common/pkg/obs"
)

func main() {
    ctx := context.Background()

    // Initialize observability with default config
    config := obs.DefaultConfig()
    config.ServiceName = "my-service"
    config.ServiceVersion = "1.0.0"
    config.Environment = "production"

    o, err := obs.Init(ctx, config)
    if err != nil {
        panic(err)
    }
    defer obs.Shutdown(ctx)

    // Use global functions
    obs.Info(ctx, "service started", "port", 8080)
    
    // Your application logic here
    doWork(ctx)
}

func doWork(ctx context.Context) {
    // Create a trace span
    tracer := obs.Tracer("my-service")
    ctx, span := tracer.Start(ctx, "do-work")
    defer span.End()

    // Log with trace correlation
    obs.Info(ctx, "starting work", "operation", "process-data")

    // Record metrics
    meter := obs.Meter("my-service")
    counter, _ := meter.Int64Counter("operations_total")
    counter.Add(ctx, 1)

    time.Sleep(100 * time.Millisecond)
    
    obs.Info(ctx, "work completed", "duration_ms", 100)
}
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVICE_NAME` | `"unknown"` | Service name for traces and logs |
| `SERVICE_VERSION` | `"dev"` | Service version |
| `ENV` | `"development"` | Environment (dev, staging, prod) |
| `OTLP_ENDPOINT` | `""` | OpenTelemetry collector endpoint |
| `OTLP_INSECURE` | `false` | Use insecure connection to OTLP |
| `OTLP_TIMEOUT` | `"30s"` | OTLP export timeout |
| `TRACING_SAMPLE_RATIO` | `1.0` | Trace sampling ratio (0.0-1.0) |
| `METRICS_ENABLED` | `true` | Enable metrics collection |
| `METRICS_PATH` | `"/metrics"` | Metrics HTTP endpoint path |
| `METRICS_PORT` | `9090` | Metrics HTTP server port |
| `LOG_LEVEL` | `"info"` | Log level (debug, info, warn, error) |
| `LOG_PRETTY` | `false` | Use pretty text format instead of JSON |
| `LOG_REDACT_TEXT` | `true` | Enable PII redaction in logs |
| `LOG_HASH_PII` | `true` | Hash redacted PII instead of masking |

### Programmatic Configuration

```go
config := obs.Config{
    ServiceName:        "my-service",
    ServiceVersion:     "1.2.3",
    Environment:        "production",
    OTLPEndpoint:       "https://otel-collector:4318",
    TracingSampleRatio: 0.1, // Sample 10% of traces
    MetricsEnabled:     true,
    MetricsPort:        9090,
    LogLevel:           "info",
    LogPretty:          false,
    ResourceAttributes: map[string]string{
        "deployment.environment": "prod",
        "service.namespace":      "backend",
    },
}
```

## Usage Examples

### 1. HTTP Server with Observability

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/quiby-ai/common/pkg/obs"
)

func main() {
    ctx := context.Background()

    // Initialize observability
    config := obs.DefaultConfig()
    config.ServiceName = "api-server"
    config.ServiceVersion = "1.0.0"
    config.OTLPEndpoint = "http://jaeger:14268/api/traces"

    _, err := obs.Init(ctx, config)
    if err != nil {
        panic(err)
    }
    defer obs.Shutdown(ctx)

    // Setup HTTP handlers
    http.HandleFunc("/api/users", handleUsers)
    http.HandleFunc("/health", handleHealth)
    
    // Expose metrics endpoint
    metricsProvider := obs.Global().MetricsProvider()
    http.Handle("/metrics", metricsProvider.Handler())

    obs.Info(ctx, "server starting", "port", 8080)
    http.ListenAndServe(":8080", nil)
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Create trace span
    tracer := obs.Tracer("api-server")
    ctx, span := tracer.Start(ctx, "handle-users")
    defer span.End()

    // Create metrics
    meter := obs.Meter("api-server")
    requestCounter, _ := meter.Int64Counter("http_requests_total")
    requestDuration, _ := meter.Float64Histogram("http_request_duration_seconds")

    start := time.Now()
    
    // Log request
    obs.Info(ctx, "handling request", 
        "method", r.Method,
        "path", r.URL.Path,
        "user_agent", r.UserAgent(),
    )

    // Simulate work
    users := getUsers(ctx)
    
    // Record metrics
    requestCounter.Add(ctx, 1)
    requestDuration.Record(ctx, time.Since(start).Seconds())

    // Log response
    obs.Info(ctx, "request completed",
        "status", 200,
        "user_count", len(users),
        "duration_ms", time.Since(start).Milliseconds(),
    )

    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"users": %d}`, len(users))
}

func getUsers(ctx context.Context) []string {
    // Create child span
    tracer := obs.Tracer("api-server")
    ctx, span := tracer.Start(ctx, "get-users")
    defer span.End()

    obs.Debug(ctx, "fetching users from database")
    
    // Simulate database call
    time.Sleep(50 * time.Millisecond)
    
    return []string{"alice", "bob", "charlie"}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
    obs.Info(r.Context(), "health check")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

### 2. Background Worker with Error Handling

```go
package main

import (
    "context"
    "errors"
    "time"

    "github.com/quiby-ai/common/pkg/obs"
)

func main() {
    ctx := context.Background()

    config := obs.DefaultConfig()
    config.ServiceName = "worker"
    config.LogLevel = "debug"

    _, err := obs.Init(ctx, config)
    if err != nil {
        panic(err)
    }
    defer obs.Shutdown(ctx)

    // Start worker
    worker := &Worker{}
    worker.Run(ctx)
}

type Worker struct{}

func (w *Worker) Run(ctx context.Context) {
    obs.Info(ctx, "worker started")

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            obs.Info(ctx, "worker shutting down")
            return
        case <-ticker.C:
            w.processJob(ctx)
        }
    }
}

func (w *Worker) processJob(ctx context.Context) {
    timer := obs.StartTimer()
    
    tracer := obs.Tracer("worker")
    ctx, span := tracer.Start(ctx, "process-job")
    defer span.End()

    jobID := "job-123"
    obs.Info(ctx, "processing job", "job_id", jobID)

    // Simulate work that might fail
    if err := w.doWork(ctx, jobID); err != nil {
        obs.Error(ctx, "job failed", err, "job_id", jobID)
        
        // Record failure metric
        meter := obs.Meter("worker")
        failureCounter, _ := meter.Int64Counter("job_failures_total")
        failureCounter.Add(ctx, 1)
        
        obs.Event(ctx, "job_processed", obs.StatusError, "job_id", jobID)
        return
    }

    duration := timer()
    obs.EventWithLatency(ctx, "job_processed", obs.StatusOK, duration, "job_id", jobID)
    
    // Record success metric
    meter := obs.Meter("worker")
    successCounter, _ := meter.Int64Counter("job_success_total")
    processingTime, _ := meter.Float64Histogram("job_duration_seconds")
    
    successCounter.Add(ctx, 1)
    processingTime.Record(ctx, duration.Seconds())
}

func (w *Worker) doWork(ctx context.Context, jobID string) error {
    obs.Debug(ctx, "starting work", "job_id", jobID)
    
    // Simulate some work
    time.Sleep(100 * time.Millisecond)
    
    // Simulate occasional failures
    if time.Now().UnixNano()%10 == 0 {
        return errors.New("random failure")
    }
    
    obs.Debug(ctx, "work completed", "job_id", jobID)
    return nil
}
```

### 3. Using with Existing Logger

```go
package main

import (
    "context"
    "log/slog"

    "github.com/quiby-ai/common/pkg/obs"
)

func main() {
    ctx := context.Background()

    config := obs.DefaultConfig()
    config.ServiceName = "my-app"
    
    o, err := obs.Init(ctx, config)
    if err != nil {
        panic(err)
    }
    defer obs.Shutdown(ctx)

    // Get the structured logger
    logger := o.Logger().Logger()
    
    // Use it directly
    logger.Info(ctx, "direct logger usage", "key", "value")
    
    // Or get logger with tracing context
    tracingLogger := o.Logger().WithTracing(ctx)
    tracingLogger.Info(ctx, "logger with tracing", "operation", "test")
}
```

## Global Functions

For convenience, the package provides global functions that work with the global observability instance:

```go
// Logging
obs.Debug(ctx, "debug message", "key", "value")
obs.Info(ctx, "info message", "key", "value")
obs.Warn(ctx, "warning message", "key", "value")
obs.Error(ctx, "error occurred", err, "key", "value")
obs.Event(ctx, "user_login", obs.StatusOK, "user_id", "123")

// Tracing
tracer := obs.Tracer("service-name")
ctx, span := tracer.Start(ctx, "operation-name")
defer span.End()

// Metrics
meter := obs.Meter("service-name")
counter, _ := meter.Int64Counter("requests_total")
counter.Add(ctx, 1)
```

## Best Practices

1. **Initialize Early**: Call `obs.Init()` at the start of your main function
2. **Always Shutdown**: Use `defer obs.Shutdown(ctx)` to ensure proper cleanup
3. **Use Context**: Pass context through your call stack for trace correlation
4. **Structured Logging**: Use key-value pairs instead of formatted strings
5. **Meaningful Span Names**: Use descriptive names for trace spans
6. **Metric Naming**: Follow Prometheus naming conventions
7. **Error Handling**: Always log errors with context and relevant attributes

## Docker Integration

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o myservice

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/myservice .

# Set observability configuration
ENV SERVICE_NAME=myservice
ENV SERVICE_VERSION=1.0.0
ENV ENV=production
ENV OTLP_ENDPOINT=http://jaeger:14268/api/traces
ENV LOG_LEVEL=info

EXPOSE 8080 9090
CMD ["./myservice"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  myservice:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"  # Metrics endpoint
    environment:
      - OTLP_ENDPOINT=http://jaeger:14268/api/traces
      - METRICS_ENABLED=true
    depends_on:
      - jaeger

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "14268:14268"  # HTTP collector
```

## Troubleshooting

### Common Issues

1. **Traces not appearing**: Check OTLP_ENDPOINT configuration and network connectivity
2. **High memory usage**: Reduce TRACING_SAMPLE_RATIO for high-traffic services
3. **Missing metrics**: Ensure METRICS_ENABLED=true and check /metrics endpoint
4. **Log PII concerns**: Enable LOG_REDACT_TEXT and LOG_HASH_PII for production

### Health Checks

```go
func healthCheck() error {
    if !obs.Global().IsInitialized() {
        return errors.New("observability not initialized")
    }
    return nil
}
```
