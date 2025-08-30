package events

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type EventBuilder[T any] interface {
	BuildEnvelope(event T, sagaID string) Envelope[any]
}

type KafkaProducer struct {
	w *kafka.Writer
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      brokers,
		Balancer:     &kafka.Hash{},
		RequiredAcks: int(kafka.RequireAll),
		Async:        false,
	})
	return &KafkaProducer{w: w}
}

func (p *KafkaProducer) Close() error {
	return p.w.Close()
}

func (p *KafkaProducer) PublishEvent(ctx context.Context, key []byte, envelope Envelope[any]) error {
	value, err := MarshalEnvelope(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	kafkaHeaders := make([]kafka.Header, 0, len(envelope.KafkaHeaders()))
	for _, h := range envelope.KafkaHeaders() {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   h.Key,
			Value: h.Value,
		})
	}

	msg := kafka.Message{
		Topic:   envelope.Type,
		Key:     key,
		Value:   value,
		Headers: kafkaHeaders,
		Time:    time.Now(),
	}
	return p.w.WriteMessages(ctx, msg)
}

func BuildEnvelope[T any](event T, eventType string, sagaID string) Envelope[any] {
	return Envelope[any]{
		MessageID:  uuid.NewString(),
		SagaID:     sagaID,
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Payload:    event,
		Meta: Meta{
			AppID:         "review-ingestor", // Set appropriate app ID
			Initiator:     InitiatorSystem,   // Default to system initiator
			Retries:       0,
			SchemaVersion: SchemaVersionV1,
		},
	}
}

// BuildEnvelopeWithMeta creates an envelope with custom meta information
func BuildEnvelopeWithMeta[T any](event T, eventType string, sagaID string, appID string, initiator Initiator) Envelope[any] {
	return Envelope[any]{
		MessageID:  uuid.NewString(),
		SagaID:     sagaID,
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Payload:    event,
		Meta: Meta{
			AppID:         appID,
			Initiator:     initiator,
			Retries:       0,
			SchemaVersion: SchemaVersionV1,
		},
	}
}
