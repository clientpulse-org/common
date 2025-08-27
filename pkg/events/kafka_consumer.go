package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type SagaMessageProcessor interface {
	Handle(ctx context.Context, payload any, sagaID string) error
}

type KafkaConsumer struct {
	reader    *kafka.Reader
	processor any
}

func NewKafkaConsumer(brokers []string, topic string, groupID string) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})
	return &KafkaConsumer{reader: reader}
}

func (kc *KafkaConsumer) SetProcessor(processor any) {
	kc.processor = processor
}

func (kc *KafkaConsumer) Run(ctx context.Context) error {
	for {
		m, err := kc.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		switch p := kc.processor.(type) {
		case SagaMessageProcessor:
			var fullMessage struct {
				SagaID  string `json:"saga_id"`
				Payload any    `json:"payload"`
			}
			if err = json.Unmarshal(m.Value, &fullMessage); err != nil {
				log.Printf("invalid saga message: %v", err)
				continue
			}
			if err = p.Handle(ctx, fullMessage.Payload, fullMessage.SagaID); err != nil {
				log.Printf("handle error: %v", err)
			}
		default:
			log.Printf("no processor set for consumer")
		}
	}
}

func (kc *KafkaConsumer) Close() error {
	if kc.reader != nil {
		return kc.reader.Close()
	}
	return nil
}
