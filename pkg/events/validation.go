package events

// ValidationError represents a validation error with field path and message.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ValidationResult contains validation results and errors.
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidateEnvelope validates the envelope structure and metadata.
func ValidateEnvelope[T any](envelope Envelope[T]) ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate required envelope fields
	if envelope.SagaID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "saga_id",
			Message: "saga_id is required",
		})
	}

	if envelope.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "type",
			Message: "type is required",
		})
	}

	if envelope.OccurredAt.IsZero() {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "occurred_at",
			Message: "occurred_at is required",
		})
	}

	// Validate meta fields
	if envelope.Meta.AppID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "meta.app_id",
			Message: "meta.app_id is required",
		})
	}

	if envelope.Meta.TenantID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "meta.tenant_id",
			Message: "meta.tenant_id is required",
		})
	}

	if envelope.Meta.Initiator == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "meta.initiator",
			Message: "meta.initiator is required",
		})
	}

	if envelope.Meta.SchemaVersion == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "meta.schema_version",
			Message: "meta.schema_version is required",
		})
	}

	return result
}
