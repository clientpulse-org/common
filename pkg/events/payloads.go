package events

import "github.com/go-playground/validator/v10"

// ExtractRequest represents the payload for pipeline.extract_reviews.request events.
type ExtractRequest struct {
	AppID     string   `json:"app_id" validate:"required"`
	AppName   string   `json:"app_name" validate:"required"`
	Countries []string `json:"countries" validate:"required,min=1,dive,len=2"`
	DateFrom  string   `json:"date_from" validate:"required,datetime=2006-01-02"`
	DateTo    string   `json:"date_to" validate:"required,datetime=2006-01-02"`
}

func (s *ExtractRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}

// ExtractCompleted represents the payload for pipeline.extract_reviews.completed events.
type ExtractCompleted struct {
	ExtractRequest
	Count int `json:"count" validate:"required,min=0"` // Number of reviews extracted
}

func (s *ExtractCompleted) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}

// PrepareRequest represents the payload for pipeline.prepare_reviews.request events.
type PrepareRequest struct {
	ExtractRequest
}

func (s *PrepareRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}

// PrepareCompleted represents the payload for pipeline.prepare_reviews.completed events.
type PrepareCompleted struct {
	PrepareRequest
	CleanCount int `json:"clean_count" validate:"required,min=0"`
}

func (s *PrepareCompleted) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}

// VectorizeRequest represents the payload for pipeline.vectorize_reviews.request events.
type VectorizeRequest struct {
	ExtractRequest
}

func (s *VectorizeRequest) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}

// VectorizeCompleted represents the payload for pipeline.vectorize_reviews.completed events.
type VectorizeCompleted struct {
	VectorizeRequest
}

func (s *VectorizeCompleted) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
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
	Step        SagaStep   `json:"step" validate:"required,oneof=extract prepare vectorize"`
	Code        FailedCode `json:"code" validate:"required,oneof=SOURCE_UNAVAILABLE RATE_LIMIT AUTH_FAILED TEMP_STORAGE_UNAVAILABLE WRITE_FAILED VALIDATION_ERROR SCHEMA_MISMATCH UNKNOWN"`
	Recoverable bool       `json:"recoverable" validate:"required"`
	// Details     string     `json:"details" validate:"omitempty"`
	// Context     any        `json:"context" validate:"omitempty"`
}

func (s *Failed) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
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
	SagaStepExtract   SagaStep = "extract"
	SagaStepPrepare   SagaStep = "prepare"
	SagaStepVectorize SagaStep = "vectorize"
)

type StateChangedContext struct {
	Message string `json:"message" validate:"required"`
}

// StateChanged represents the payload for saga.orchestrator.state.changed events.
type StateChanged struct {
	Status  SagaStatus          `json:"status" validate:"required,oneof=running failed completed"`
	Step    SagaStep            `json:"step" validate:"required,oneof=extract prepare vectorize"`
	Context StateChangedContext `json:"context" validate:"required"`
	Error   *struct {
		Code    FailedCode `json:"code" validate:"required,oneof=SOURCE_UNAVAILABLE RATE_LIMIT AUTH_FAILED TEMP_STORAGE_UNAVAILABLE WRITE_FAILED VALIDATION_ERROR SCHEMA_MISMATCH UNKNOWN"`
		Message string     `json:"message" validate:"omitempty"`
	} `json:"error,omitempty"`
}

func (s *StateChanged) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}
