package obs

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type LoggingProvider struct {
	logger *Logger
	config Config
}

func newLoggingProvider(config Config) (*LoggingProvider, error) {
	logger := initLogger(config)

	return &LoggingProvider{
		logger: logger,
		config: config,
	}, nil
}

func (lp *LoggingProvider) Logger() *Logger {
	return lp.logger
}

func (lp *LoggingProvider) WithTracing(ctx context.Context) *Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return lp.logger
	}

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()

	ctxWithCorrelation := withCorrelation(ctx, traceID, spanID, "", "", "", "")
	return lp.logger.withContext(ctxWithCorrelation)
}

func (lp *LoggingProvider) Debug(ctx context.Context, msg string, attrs ...any) {
	logger := lp.WithTracing(ctx)
	logger.Debug(ctx, msg, attrs...)
}

func (lp *LoggingProvider) Info(ctx context.Context, msg string, attrs ...any) {
	logger := lp.WithTracing(ctx)
	logger.Info(ctx, msg, attrs...)
}

func (lp *LoggingProvider) Warn(ctx context.Context, msg string, attrs ...any) {
	logger := lp.WithTracing(ctx)
	logger.Warn(ctx, msg, attrs...)
}

func (lp *LoggingProvider) Error(ctx context.Context, msg string, err error, attrs ...any) {
	logger := lp.WithTracing(ctx)
	logger.Error(ctx, msg, err, attrs...)
}

func (lp *LoggingProvider) Event(ctx context.Context, event, status string, attrs ...any) {
	logger := lp.WithTracing(ctx)
	logger.Event(ctx, event, status, attrs...)
}

func (lp *LoggingProvider) Shutdown(ctx context.Context) error {
	return nil
}

func Debug(ctx context.Context, msg string, attrs ...any) {
	if globalObs != nil && globalObs.logging != nil {
		globalObs.logging.Debug(ctx, msg, attrs...)
	}
}

func Info(ctx context.Context, msg string, attrs ...any) {
	if globalObs != nil && globalObs.logging != nil {
		globalObs.logging.Info(ctx, msg, attrs...)
	}
}

func Warn(ctx context.Context, msg string, attrs ...any) {
	if globalObs != nil && globalObs.logging != nil {
		globalObs.logging.Warn(ctx, msg, attrs...)
	}
}

func Error(ctx context.Context, msg string, err error, attrs ...any) {
	if globalObs != nil && globalObs.logging != nil {
		globalObs.logging.Error(ctx, msg, err, attrs...)
	}
}

func Event(ctx context.Context, event, status string, attrs ...any) {
	if globalObs != nil && globalObs.logging != nil {
		globalObs.logging.Event(ctx, event, status, attrs...)
	}
}
