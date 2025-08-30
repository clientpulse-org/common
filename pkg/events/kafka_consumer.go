package events

import (
	"context"
	"encoding/json"
	"fmt"
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

// NewTypedKafkaConsumer creates a consumer that can handle specific event types with proper validation
func NewTypedKafkaConsumer(brokers []string, topic string, groupID string) *KafkaConsumer {
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
			// First, try to unmarshal as a raw envelope to get basic structure
			var rawEnvelope map[string]json.RawMessage
			if err = json.Unmarshal(m.Value, &rawEnvelope); err != nil {
				log.Printf("invalid message format: %v", err)
				continue
			}

			// Extract saga_id and type for validation
			var sagaID string
			if sagaIDRaw, exists := rawEnvelope["saga_id"]; exists {
				if err = json.Unmarshal(sagaIDRaw, &sagaID); err != nil {
					log.Printf("invalid saga_id format: %v", err)
					continue
				}
			} else {
				log.Printf("missing saga_id in message")
				continue
			}

			var eventType string
			if typeRaw, exists := rawEnvelope["type"]; exists {
				if err = json.Unmarshal(typeRaw, &eventType); err != nil {
					log.Printf("invalid type format: %v", err)
					continue
				}
			} else {
				log.Printf("missing type in message")
				continue
			}

			// Extract and validate payload based on event type
			payload, err := kc.extractAndValidatePayload(rawEnvelope, eventType)
			if err != nil {
				log.Printf("payload validation failed: %v", err)
				continue
			}

			// Log message info for debugging
			kc.LogMessageInfo(sagaID, eventType, payload)

			// Process the message
			if err = p.Handle(ctx, payload, sagaID); err != nil {
				log.Printf("handle error: %v", err)
			}
		default:
			log.Printf("no processor set for consumer")
		}
	}
}

// ValidateMessage validates the entire message envelope before processing
func (kc *KafkaConsumer) ValidateMessage(data []byte) (ValidationResult, error) {
	var envelope Envelope[any]
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ValidationResult{Valid: false}, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}

	return ValidateEnvelope(envelope), nil
}

// LogMessageInfo logs message information for debugging
func (kc *KafkaConsumer) LogMessageInfo(sagaID, eventType string, payload any) {
	log.Printf("Processing message - SagaID: %s, Type: %s, Payload: %+v", sagaID, eventType, payload)
}

// extractAndValidatePayload extracts and validates the payload based on the event type
func (kc *KafkaConsumer) extractAndValidatePayload(rawEnvelope map[string]json.RawMessage, eventType string) (any, error) {
	payloadRaw, exists := rawEnvelope["payload"]
	if !exists {
		return nil, fmt.Errorf("missing payload in message")
	}

	// Determine the expected payload type based on event type
	var payload any
	switch eventType {
	case PipelineExtractRequest:
		var extractReq ExtractRequest
		if err := json.Unmarshal(payloadRaw, &extractReq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ExtractRequest: %w", err)
		}
		if err := extractReq.Validate(); err != nil {
			return nil, fmt.Errorf("ExtractRequest validation failed: %w", err)
		}
		payload = extractReq

	case PipelineExtractCompleted:
		var extractCompleted ExtractCompleted
		if err := json.Unmarshal(payloadRaw, &extractCompleted); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ExtractCompleted: %w", err)
		}
		if err := extractCompleted.Validate(); err != nil {
			return nil, fmt.Errorf("ExtractCompleted validation failed: %w", err)
		}
		payload = extractCompleted

	case PipelinePrepareRequest:
		var prepareReq PrepareRequest
		if err := json.Unmarshal(payloadRaw, &prepareReq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal PrepareRequest: %w", err)
		}
		if err := prepareReq.Validate(); err != nil {
			return nil, fmt.Errorf("PrepareRequest validation failed: %w", err)
		}
		payload = prepareReq

	case PipelinePrepareCompleted:
		var prepareCompleted PrepareCompleted
		if err := json.Unmarshal(payloadRaw, &prepareCompleted); err != nil {
			return nil, fmt.Errorf("failed to unmarshal PrepareCompleted: %w", err)
		}
		if err := prepareCompleted.Validate(); err != nil {
			return nil, fmt.Errorf("PrepareCompleted validation failed: %w", err)
		}
		payload = prepareCompleted

	case PipelineVectorizeRequest:
		var vectorizeReq VectorizeRequest
		if err := json.Unmarshal(payloadRaw, &vectorizeReq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal VectorizeRequest: %w", err)
		}
		if err := vectorizeReq.Validate(); err != nil {
			return nil, fmt.Errorf("VectorizeRequest validation failed: %w", err)
		}
		payload = vectorizeReq

	case PipelineVectorizeCompleted:
		var vectorizeCompleted VectorizeCompleted
		if err := json.Unmarshal(payloadRaw, &vectorizeCompleted); err != nil {
			return nil, fmt.Errorf("failed to unmarshal VectorizeCompleted: %w", err)
		}
		if err := vectorizeCompleted.Validate(); err != nil {
			return nil, fmt.Errorf("VectorizeCompleted validation failed: %w", err)
		}
		payload = vectorizeCompleted

	case PipelineFailed:
		var failed Failed
		if err := json.Unmarshal(payloadRaw, &failed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Failed: %w", err)
		}
		if err := failed.Validate(); err != nil {
			return nil, fmt.Errorf("Failed validation failed: %w", err)
		}
		payload = failed

	case SagaStateChanged:
		var stateChanged StateChanged
		if err := json.Unmarshal(payloadRaw, &stateChanged); err != nil {
			return nil, fmt.Errorf("failed to unmarshal StateChanged: %w", err)
		}
		if err := stateChanged.Validate(); err != nil {
			return nil, fmt.Errorf("StateChanged validation failed: %w", err)
		}
		payload = stateChanged

	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	return payload, nil
}

func (kc *KafkaConsumer) Close() error {
	if kc.reader != nil {
		return kc.reader.Close()
	}
	return nil
}
