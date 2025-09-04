package obs

import "errors"

var (
	ErrInvalidServiceName = errors.New("service name cannot be empty")
	ErrInvalidSampleRatio = errors.New("tracing sample ratio must be between 0 and 1")
	ErrInvalidMetricsPort = errors.New("metrics port must be between 1 and 65535")
	ErrAlreadyInitialized = errors.New("observability already initialized")
	ErrNotInitialized     = errors.New("observability not initialized")
	ErrTracingInitFailed  = errors.New("failed to initialize tracing")
	ErrMetricsInitFailed  = errors.New("failed to initialize metrics")
	ErrLoggingInitFailed  = errors.New("failed to initialize logging")
	ErrShutdownTimeout    = errors.New("shutdown timeout exceeded")
	ErrShutdownFailed     = errors.New("shutdown failed")
)
