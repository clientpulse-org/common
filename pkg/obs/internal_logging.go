package obs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"
)

type contextKey string

const (
	traceIDKey   contextKey = "trace_id"
	spanIDKey    contextKey = "span_id"
	sagaIDKey    contextKey = "saga_id"
	messageIDKey contextKey = "messageKey"
	reviewIDKey  contextKey = "review_id"
	appIDKey     contextKey = "app_id"

	StatusOK       = "ok"
	StatusError    = "error"
	StatusRetrying = "retrying"
	StatusSkipped  = "skipped"

	ErrKindValidation   = "validation"
	ErrKindNotFound     = "not_found"
	ErrKindUnauthorized = "unauthorized"
	ErrKindForbidden    = "forbidden"
	ErrKindConflict     = "conflict"
	ErrKindTimeout      = "timeout"
	ErrKindInternal     = "internal"
	ErrKindExternal     = "external"
	ErrKindNetwork      = "network"
	ErrKindDatabase     = "database"
	ErrKindKafka        = "kafka"
	ErrKindHTTP         = "http"
	ErrKindGRPC         = "grpc"
)

var (
	piiPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(password|secret|token|key|auth|credential)\s*[:=]\s*["']?[^"'\s]+["']?`),
		regexp.MustCompile(`(?i)(email)\s*[:=]\s*["']?[^"'\s@]+@[^"'\s]+\.[^"'\s]+["']?`),
		regexp.MustCompile(`(?i)(phone|mobile|tel)\s*[:=]\s*["']?[\d\-\+\(\)\s]+["']?`),
		regexp.MustCompile(`(?i)(ssn|social|credit|card)\s*[:=]\s*["']?[\d\-\s]+["']?`),
		regexp.MustCompile(`(?i)(ip|address)\s*[:=]\s*["']?[\d\.]+["']?`),
	}
)

type Logger struct {
	*slog.Logger
	config *loggingConfig
}

type loggingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	LogLevel       string
	LogPretty      bool
	LogRedactText  bool
	LogHashPII     bool
}

func initLogger(config Config) *Logger {
	loggingConfig := &loggingConfig{
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
		Environment:    config.Environment,
		LogLevel:       config.LogLevel,
		LogPretty:      config.LogPretty,
		LogRedactText:  config.LogRedactText,
		LogHashPII:     config.LogHashPII,
	}

	level := parseLogLevel(loggingConfig.LogLevel)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format(time.RFC3339Nano))
			}
			return a
		},
	}

	var handler slog.Handler
	if loggingConfig.LogPretty {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	hostname, _ := os.Hostname()
	gitSHA := getGitSHA()

	defaultAttrs := []any{
		"service", loggingConfig.ServiceName,
		"version", loggingConfig.ServiceVersion,
		"env", loggingConfig.Environment,
		"hostname", hostname,
		"git_sha", gitSHA,
	}

	return &Logger{
		Logger: logger.With(defaultAttrs...),
		config: loggingConfig,
	}
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getGitSHA() string {
	if sha := os.Getenv("GIT_SHA"); sha != "" {
		return sha
	}
	if sha := os.Getenv("COMMIT_SHA"); sha != "" {
		return sha
	}
	return "unknown"
}

func withCorrelation(ctx context.Context, traceID, spanID, sagaID, messageID, reviewID, appID string) context.Context {
	if traceID != "" {
		ctx = context.WithValue(ctx, traceIDKey, traceID)
	}
	if spanID != "" {
		ctx = context.WithValue(ctx, spanIDKey, spanID)
	}
	if sagaID != "" {
		ctx = context.WithValue(ctx, sagaIDKey, sagaID)
	}
	if messageID != "" {
		ctx = context.WithValue(ctx, messageIDKey, messageID)
	}
	if reviewID != "" {
		ctx = context.WithValue(ctx, reviewIDKey, reviewID)
	}
	if appID != "" {
		ctx = context.WithValue(ctx, appIDKey, appID)
	}
	return ctx
}

func (l *Logger) withContext(ctx context.Context) *Logger {
	attrs := []any{}
	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		attrs = append(attrs, "trace_id", traceID)
	}
	if spanID, ok := ctx.Value(spanIDKey).(string); ok && spanID != "" {
		attrs = append(attrs, "span_id", spanID)
	}
	if sagaID, ok := ctx.Value(sagaIDKey).(string); ok && sagaID != "" {
		attrs = append(attrs, "saga_id", sagaID)
	}
	if messageID, ok := ctx.Value(messageIDKey).(string); ok && messageID != "" {
		attrs = append(attrs, "message_id", messageID)
	}
	if reviewID, ok := ctx.Value(reviewIDKey).(string); ok && reviewID != "" {
		attrs = append(attrs, "review_id", reviewID)
	}
	if appID, ok := ctx.Value(appIDKey).(string); ok && appID != "" {
		attrs = append(attrs, "app_id", appID)
	}

	if len(attrs) == 0 {
		return l
	}

	return &Logger{
		Logger: l.With(attrs...),
		config: l.config,
	}
}

func (l *Logger) redactPII(msg string) string {
	if !l.config.LogRedactText {
		return msg
	}

	redacted := msg
	for _, pattern := range piiPatterns {
		redacted = pattern.ReplaceAllStringFunc(redacted, func(match string) string {
			if l.config.LogHashPII {
				hash := sha256.Sum256([]byte(match))
				return fmt.Sprintf("[REDACTED:%s]", hex.EncodeToString(hash[:8]))
			}
			return "[REDACTED]"
		})
	}
	return redacted
}

func (l *Logger) processAttrs(attrs []any) []any {
	if !l.config.LogRedactText {
		return attrs
	}

	processed := make([]any, len(attrs))
	copy(processed, attrs)

	for i := 0; i < len(processed); i += 2 {
		if i+1 < len(processed) {
			key, ok := processed[i].(string)
			if !ok {
				continue
			}

			value, ok := processed[i+1].(string)
			if !ok {
				continue
			}

			for _, pattern := range piiPatterns {
				if pattern.MatchString(fmt.Sprintf("%s: %s", key, value)) {
					if l.config.LogHashPII {
						hash := sha256.Sum256([]byte(value))
						processed[i+1] = fmt.Sprintf("[REDACTED:%s]", hex.EncodeToString(hash[:8]))
					} else {
						processed[i+1] = "[REDACTED]"
					}
					break
				}
			}
		}
	}

	return processed
}

func (l *Logger) Log(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	msg = l.redactPII(msg)
	attrs = l.processAttrs(attrs)
	l.Logger.Log(ctx, level, msg, attrs...)
}

func (l *Logger) Debug(ctx context.Context, msg string, attrs ...any) {
	l.Log(ctx, slog.LevelDebug, msg, attrs...)
}

func (l *Logger) Info(ctx context.Context, msg string, attrs ...any) {
	l.Log(ctx, slog.LevelInfo, msg, attrs...)
}

func (l *Logger) Warn(ctx context.Context, msg string, attrs ...any) {
	l.Log(ctx, slog.LevelWarn, msg, attrs...)
}

func (l *Logger) Error(ctx context.Context, msg string, err error, attrs ...any) {
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}
	l.Log(ctx, slog.LevelError, msg, attrs...)
}

func (l *Logger) Event(ctx context.Context, event, status string, attrs ...any) {
	attrs = append([]any{"event", event, "status", status}, attrs...)
	l.Info(ctx, event, attrs...)
}

func (l *Logger) EventWithLatency(ctx context.Context, event, status string, latency time.Duration, attrs ...any) {
	attrs = append([]any{
		"event", event,
		"status", status,
		"latency_ms", latency.Milliseconds(),
	}, attrs...)
	l.Info(ctx, event, attrs...)
}

func StartTimer() func() time.Duration {
	start := time.Now()
	return func() time.Duration {
		return time.Since(start)
	}
}
