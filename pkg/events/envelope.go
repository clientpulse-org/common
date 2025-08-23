package events

import (
	"encoding/json"
	"time"
)

const SchemaVersionV1 = "v1"

type Initiator string

const (
	InitiatorUser   Initiator = "user"
	InitiatorSystem Initiator = "system"
)

// Meta holds auxiliary metadata not part of the core payload.
type Meta struct {
	AppID         string    `json:"app_id"`
	Initiator     Initiator `json:"initiator"`
	Retries       int       `json:"retries"`
	SchemaVersion string    `json:"schema_version"`
}

// Envelope defines the standard message envelope used for all events.
//
// MessageID and TraceID are optional. SagaID is required.
// OccurredAt is serialized in RFC3339 UTC by the standard library.
type Envelope[T any] struct {
	MessageID  string    `json:"message_id,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	SagaID     string    `json:"saga_id"`
	Type       string    `json:"type"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    T         `json:"payload"`
	Meta       Meta      `json:"meta"`
}

// MarshalEnvelope serializes the envelope to JSON.
func MarshalEnvelope[T any](e Envelope[T]) ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalEnvelope deserializes the envelope from JSON into the provided payload type.
func UnmarshalEnvelope[T any](data []byte) (Envelope[T], error) {
	var e Envelope[T]
	if err := json.Unmarshal(data, &e); err != nil {
		return e, err
	}
	return e, nil
}
