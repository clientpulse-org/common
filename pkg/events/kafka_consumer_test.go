package events

import (
	"context"
	"fmt"
	"testing"
)

type MockMessageProcessor struct {
	processedMessages [][]byte
	shouldError       bool
}

func (m *MockMessageProcessor) Handle(ctx context.Context, message []byte) error {
	if m.shouldError {
		return fmt.Errorf("mock error")
	}
	m.processedMessages = append(m.processedMessages, message)
	return nil
}

type MockSagaMessageProcessor struct {
	handledMessages []struct {
		payload any
		sagaID  string
	}
	shouldError bool
}

func (m *MockSagaMessageProcessor) Handle(ctx context.Context, payload any, sagaID string) error {
	if m.shouldError {
		return fmt.Errorf("mock saga error")
	}
	m.handledMessages = append(m.handledMessages, struct {
		payload interface{}
		sagaID  string
	}{payload: payload, sagaID: sagaID})
	return nil
}

func TestNewKafkaConsumer(t *testing.T) {
	brokers := []string{"localhost:9092"}
	topic := "test-topic"
	groupID := "test-group"

	consumer := NewKafkaConsumer(brokers, topic, groupID)
	if consumer == nil {
		t.Fatal("NewKafkaConsumer returned nil")
	}
	if consumer.reader == nil {
		t.Fatal("Kafka reader is nil")
	}
}

func TestSetProcessor(t *testing.T) {
	consumer := NewKafkaConsumer([]string{"localhost:9092"}, "test", "test")
	processor := &MockMessageProcessor{}

	consumer.SetProcessor(processor)
	if consumer.processor != processor {
		t.Fatal("Processor not set correctly")
	}
}

func TestSagaMessageProcessorInterface(t *testing.T) {
	consumer := NewKafkaConsumer([]string{"localhost:9092"}, "test", "test")
	processor := &MockSagaMessageProcessor{shouldError: false}
	consumer.SetProcessor(processor)

	var _ SagaMessageProcessor = processor
}
