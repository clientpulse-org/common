package events

import (
	"testing"
	"time"
)

func TestNewKafkaProducer(t *testing.T) {
	brokers := []string{"localhost:9092"}
	producer := NewKafkaProducer(brokers)

	if producer == nil {
		t.Fatal("NewKafkaProducer returned nil")
	}
	if producer.w == nil {
		t.Fatal("Kafka writer is nil")
	}
}

func TestBuildEnvelope(t *testing.T) {
	event := "test-event"
	eventType := "test.type"
	sagaID := "test-saga-123"

	envelope := BuildEnvelope(event, eventType, sagaID)

	if envelope.MessageID == "" {
		t.Error("MessageID should not be empty")
	}
	if envelope.SagaID != sagaID {
		t.Errorf("Expected SagaID %s, got %s", sagaID, envelope.SagaID)
	}
	if envelope.Type != eventType {
		t.Errorf("Expected Type %s, got %s", eventType, envelope.Type)
	}
	if envelope.Payload != event {
		t.Errorf("Expected Payload %v, got %v", event, envelope.Payload)
	}
	if envelope.Meta.SchemaVersion != SchemaVersionV1 {
		t.Errorf("Expected SchemaVersion %s, got %s", SchemaVersionV1, envelope.Meta.SchemaVersion)
	}

	// Check that OccurredAt is recent (within last second)
	now := time.Now().UTC()
	if envelope.OccurredAt.After(now) || envelope.OccurredAt.Before(now.Add(-time.Second)) {
		t.Errorf("OccurredAt should be recent, got %v", envelope.OccurredAt)
	}
}

func TestProducerClose(t *testing.T) {
	producer := NewKafkaProducer([]string{"localhost:9092"})

	err := producer.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}
