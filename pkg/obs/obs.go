package obs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Observability struct {
	config       Config
	tracing      *TracingProvider
	metrics      *MetricsProvider
	logging      *LoggingProvider
	initOnce     sync.Once
	initErr      error
	shutdownOnce sync.Once
	isShutdown   bool
	mu           sync.RWMutex
}

var (
	globalObs *Observability
	globalMu  sync.RWMutex
)

func Init(ctx context.Context, config Config) (*Observability, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	globalMu.Lock()
	if globalObs != nil {
		existing := globalObs
		globalMu.Unlock()
		return existing, nil
	}
	globalMu.Unlock()

	obs := &Observability{
		config: config,
	}

	var initErr error
	obs.initOnce.Do(func() {
		obs.logging, initErr = newLoggingProvider(config)
		if initErr != nil {
			initErr = fmt.Errorf("%w: %v", ErrLoggingInitFailed, initErr)
			return
		}

		obs.tracing, initErr = newTracingProvider(ctx, config)
		if initErr != nil {
			initErr = fmt.Errorf("%w: %v", ErrTracingInitFailed, initErr)
			return
		}

		obs.metrics, initErr = newMetricsProvider(ctx, config)
		if initErr != nil {
			initErr = fmt.Errorf("%w: %v", ErrMetricsInitFailed, initErr)
			return
		}

		obs.logging.Info(ctx, "observability initialized",
			"service", config.ServiceName,
			"version", config.ServiceVersion,
			"environment", config.Environment,
			"otlp_endpoint", config.OTLPEndpoint,
			"metrics_enabled", config.MetricsEnabled,
		)
	})

	if initErr != nil {
		obs.initErr = initErr
		return nil, initErr
	}

	globalMu.Lock()
	if globalObs == nil {
		globalObs = obs
	} else {
		obs = globalObs
	}
	globalMu.Unlock()

	return obs, nil
}

func Global() *Observability {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalObs
}

func MustInit(ctx context.Context, config Config) *Observability {
	obs, err := Init(ctx, config)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize observability: %v", err))
	}
	return obs
}

func (o *Observability) Shutdown(ctx context.Context) error {
	var shutdownErr error

	o.shutdownOnce.Do(func() {
		o.mu.Lock()
		defer o.mu.Unlock()

		if o.initErr != nil {
			shutdownErr = o.initErr
			return
		}

		if o.isShutdown {
			shutdownErr = fmt.Errorf("already shutdown")
			return
		}

		var errors []error

		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if o.tracing != nil {
			if err := o.tracing.ForceFlush(shutdownCtx); err != nil {
				errors = append(errors, fmt.Errorf("failed to flush traces: %w", err))
			}
			if err := o.tracing.Shutdown(shutdownCtx); err != nil {
				errors = append(errors, fmt.Errorf("failed to shutdown tracing: %w", err))
			}
		}

		if o.metrics != nil {
			if err := o.metrics.ForceFlush(shutdownCtx); err != nil {
				errors = append(errors, fmt.Errorf("failed to flush metrics: %w", err))
			}
			if err := o.metrics.Shutdown(shutdownCtx); err != nil {
				errors = append(errors, fmt.Errorf("failed to shutdown metrics: %w", err))
			}
		}

		if o.logging != nil {
			if err := o.logging.Shutdown(shutdownCtx); err != nil {
				errors = append(errors, fmt.Errorf("failed to shutdown logging: %w", err))
			}
		}

		o.isShutdown = true

		if len(errors) > 0 {
			shutdownErr = fmt.Errorf("%w: %v", ErrShutdownFailed, errors)
			return
		}

		if o.logging != nil {
			o.logging.Info(shutdownCtx, "observability shutdown completed")
		}
	})

	return shutdownErr
}

func Shutdown(ctx context.Context) error {
	globalMu.RLock()
	obs := globalObs
	globalMu.RUnlock()

	if obs == nil {
		return ErrNotInitialized
	}

	return obs.Shutdown(ctx)
}

func (o *Observability) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	if o.tracing == nil {
		return trace.NewNoopTracerProvider().Tracer(name, opts...)
	}
	return o.tracing.Tracer(name, opts...)
}

func (o *Observability) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	if o.metrics == nil {
		return otel.Meter(name, opts...)
	}
	return o.metrics.Meter(name, opts...)
}

func (o *Observability) Logger() *LoggingProvider {
	return o.logging
}

func (o *Observability) TracingProvider() *TracingProvider {
	return o.tracing
}

func (o *Observability) MetricsProvider() *MetricsProvider {
	return o.metrics
}

func (o *Observability) LoggingProvider() *LoggingProvider {
	return o.logging
}

func (o *Observability) Config() Config {
	return o.config
}

func (o *Observability) IsInitialized() bool {
	return o.initErr == nil
}

func Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	globalMu.RLock()
	obs := globalObs
	globalMu.RUnlock()

	if obs == nil {
		return trace.NewNoopTracerProvider().Tracer(name, opts...)
	}
	return obs.Tracer(name, opts...)
}

func Meter(name string, opts ...metric.MeterOption) metric.Meter {
	globalMu.RLock()
	obs := globalObs
	globalMu.RUnlock()

	if obs == nil {
		return otel.Meter(name, opts...)
	}
	return obs.Meter(name, opts...)
}
