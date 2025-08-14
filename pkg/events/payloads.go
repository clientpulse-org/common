package events

// ExtractRequest represents the payload for pipeline.extract_reviews.request events.
type ExtractRequest struct {
	AppID     string   `json:"app_id"`
	AppName   string   `json:"app_name"`
	Countries []string `json:"countries"` // ISO-2 country codes
	DateFrom  string   `json:"date_from"` // YYYY-MM-DD format
	DateTo    string   `json:"date_to"`   // YYYY-MM-DD format
}

// ExtractCompleted represents the payload for pipeline.extract_reviews.completed events.
type ExtractCompleted struct {
	ExtractRequest
	Count int `json:"count"` // Number of reviews extracted
}

// PrepareRequest represents the payload for pipeline.prepare_reviews.request events.
// It's an alias to ExtractRequest as they share the same structure.
type PrepareRequest = ExtractRequest

// PrepareCompleted represents the payload for pipeline.prepare_reviews.completed events.
type PrepareCompleted struct {
	ExtractRequest
	Count      int `json:"count"`       // Total number of reviews
	CleanCount int `json:"clean_count"` // Number of clean/processed reviews
}

// FailedCode represents the error codes for pipeline.failed events.
type FailedCode string

const (
	FailedCodeSourceUnavailable      FailedCode = "SOURCE_UNAVAILABLE"
	FailedCodeRateLimit              FailedCode = "RATE_LIMIT"
	FailedCodeAuthFailed             FailedCode = "AUTH_FAILED"
	FailedCodeTempStorageUnavailable FailedCode = "TEMP_STORAGE_UNAVAILABLE"
	FailedCodeWriteFailed            FailedCode = "WRITE_FAILED"
	FailedCodeValidationError        FailedCode = "VALIDATION_ERROR"
	FailedCodeSchemaMismatch         FailedCode = "SCHEMA_MISMATCH"
	FailedCodeUnknown                FailedCode = "UNKNOWN"
)

// Failed represents the payload for pipeline.failed events.
type Failed struct {
	Step        string     `json:"step"`        // "extract" or "prepare"
	Code        FailedCode `json:"code"`        // Error code enum
	Recoverable bool       `json:"recoverable"` // Whether the error is recoverable
	Details     string     `json:"details"`     // Human-readable error details
	Context     struct {
		AppID     string   `json:"app_id"`
		Countries []string `json:"countries"`
	} `json:"context"`
}

// SagaStatus represents the status of a saga.
type SagaStatus string

const (
	SagaStatusRunning   SagaStatus = "running"
	SagaStatusFailed    SagaStatus = "failed"
	SagaStatusCompleted SagaStatus = "completed"
)

// SagaStep represents the current step in a saga.
type SagaStep string

const (
	SagaStepExtract SagaStep = "extract"
	SagaStepPrepare SagaStep = "prepare"
)

// StateChanged represents the payload for saga.orchestrator.state.changed events.
type StateChanged struct {
	Status  SagaStatus `json:"status"` // Current saga status
	Step    SagaStep   `json:"step"`   // Current step
	Context struct {
		Message string `json:"message"`
	} `json:"context"`
	Error *struct {
		Code    FailedCode `json:"code"`
		Message string     `json:"message"`
	} `json:"error,omitempty"` // Only present when status is "failed"
}
