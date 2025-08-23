package events

import (
	"time"
)

// NewEnvelope creates a new envelope with the given payload and metadata.
func NewEnvelope[T any](sagaID, eventType string, payload T, meta Meta) Envelope[T] {
	return Envelope[T]{
		SagaID:     sagaID,
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
		Meta:       meta,
	}
}

// NewMeta creates a new Meta struct with the required fields.
func NewMeta(appID string, initiator Initiator) Meta {
	return Meta{
		AppID:         appID,
		Initiator:     initiator,
		Retries:       0,
		SchemaVersion: SchemaVersionV1,
	}
}

// WithMessageID adds a message ID to the envelope for idempotency.
func (e Envelope[T]) WithMessageID(messageID string) Envelope[T] {
	e.MessageID = messageID
	return e
}

// WithTraceID adds a trace ID to the envelope for distributed tracing.
func (e Envelope[T]) WithTraceID(traceID string) Envelope[T] {
	e.TraceID = traceID
	return e
}

// IncrementRetries increments the retry count in the meta field.
func (e Envelope[T]) IncrementRetries() Envelope[T] {
	e.Meta.Retries++
	return e
}
