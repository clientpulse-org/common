package events

import "fmt"

// KafkaHeader represents a Kafka message header.
type KafkaHeader struct {
	Key   string
	Value []byte
}

// KafkaHeaders returns Kafka headers for envelope fields.
func (e Envelope[T]) KafkaHeaders() []KafkaHeader {
	headers := []KafkaHeader{
		{Key: "saga_id", Value: []byte(e.SagaID)},
		{Key: "event_type", Value: []byte(e.Type)},
		{Key: "tenant_id", Value: []byte(e.Meta.TenantID)},
		{Key: "app_id", Value: []byte(e.Meta.AppID)},
		{Key: "initiator", Value: []byte(string(e.Meta.Initiator))},
		{Key: "schema_version", Value: []byte(e.Meta.SchemaVersion)},
		{Key: "retries", Value: []byte(fmt.Sprintf("%d", e.Meta.Retries))},
	}

	if e.MessageID != "" {
		headers = append(headers, KafkaHeader{Key: "message_id", Value: []byte(e.MessageID)})
	}

	if e.TraceID != "" {
		headers = append(headers, KafkaHeader{Key: "trace_id", Value: []byte(e.TraceID)})
	}

	return headers
}
