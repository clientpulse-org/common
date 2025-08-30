package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockProcessor implements SagaMessageProcessor for testing
type MockProcessor struct {
	handledPayloads []any
	handledSagaIDs  []string
	shouldError     bool
}

func (m *MockProcessor) Handle(ctx context.Context, payload any, sagaID string) error {
	m.handledPayloads = append(m.handledPayloads, payload)
	m.handledSagaIDs = append(m.handledSagaIDs, sagaID)

	if m.shouldError {
		return assert.AnError
	}
	return nil
}

func TestKafkaConsumer_ExtractAndValidatePayload(t *testing.T) {
	consumer := &KafkaConsumer{}

	tests := []struct {
		name        string
		eventType   string
		payload     any
		expectError bool
	}{
		{
			name:      "valid ExtractRequest",
			eventType: PipelineExtractRequest,
			payload: ExtractRequest{
				AppID:     "test-app",
				AppName:   "Test App",
				Countries: []string{"US", "GB"},
				DateFrom:  "2024-01-01",
				DateTo:    "2024-01-31",
			},
			expectError: false,
		},
		{
			name:      "valid ExtractCompleted",
			eventType: PipelineExtractCompleted,
			payload: ExtractCompleted{
				ExtractRequest: ExtractRequest{
					AppID:     "test-app",
					AppName:   "Test App",
					Countries: []string{"US"},
					DateFrom:  "2024-01-01",
					DateTo:    "2024-01-31",
				},
				Count: 100,
			},
			expectError: false,
		},
		{
			name:      "valid PrepareRequest",
			eventType: PipelinePrepareRequest,
			payload: PrepareRequest{
				ExtractRequest: ExtractRequest{
					AppID:     "test-app",
					AppName:   "Test App",
					Countries: []string{"US"},
					DateFrom:  "2024-01-01",
					DateTo:    "2024-01-31",
				},
			},
			expectError: false,
		},
		{
			name:      "valid PrepareCompleted",
			eventType: PipelinePrepareCompleted,
			payload: PrepareCompleted{
				PrepareRequest: PrepareRequest{
					ExtractRequest: ExtractRequest{
						AppID:     "test-app",
						AppName:   "Test App",
						Countries: []string{"US"},
						DateFrom:  "2024-01-01",
						DateTo:    "2024-01-31",
					},
				},
				CleanCount: 95,
			},
			expectError: false,
		},
		{
			name:      "valid VectorizeRequest",
			eventType: PipelineVectorizeRequest,
			payload: VectorizeRequest{
				ExtractRequest: ExtractRequest{
					AppID:     "test-app",
					AppName:   "Test App",
					Countries: []string{"US"},
					DateFrom:  "2024-01-01",
					DateTo:    "2024-01-31",
				},
			},
			expectError: false,
		},
		{
			name:      "valid VectorizeCompleted",
			eventType: PipelineVectorizeCompleted,
			payload: VectorizeCompleted{
				VectorizeRequest: VectorizeRequest{
					ExtractRequest: ExtractRequest{
						AppID:     "test-app",
						AppName:   "Test App",
						Countries: []string{"US"},
						DateFrom:  "2024-01-01",
						DateTo:    "2024-01-31",
					},
				},
			},
			expectError: false,
		},
		{
			name:      "valid Failed",
			eventType: PipelineFailed,
			payload: Failed{
				Step:        SagaStepExtract,
				Code:        FailedCodeRateLimit,
				Recoverable: true,
			},
			expectError: false,
		},
		{
			name:      "valid StateChanged",
			eventType: SagaStateChanged,
			payload: StateChanged{
				Status: SagaStatusRunning,
				Step:   SagaStepExtract,
				Context: StateChangedContext{
					Message: "Starting extraction",
				},
			},
			expectError: false,
		},
		{
			name:        "unknown event type",
			eventType:   "unknown.event.type",
			payload:     "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a raw envelope with the test payload
			rawEnvelope := map[string]json.RawMessage{
				"payload": mustMarshal(tt.payload),
			}

			payload, err := consumer.extractAndValidatePayload(rawEnvelope, tt.eventType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, payload)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, payload)

				// Verify the payload type matches
				switch tt.eventType {
				case PipelineExtractRequest:
					assert.IsType(t, ExtractRequest{}, payload)
				case PipelineExtractCompleted:
					assert.IsType(t, ExtractCompleted{}, payload)
				case PipelinePrepareRequest:
					assert.IsType(t, PrepareRequest{}, payload)
				case PipelinePrepareCompleted:
					assert.IsType(t, PrepareCompleted{}, payload)
				case PipelineVectorizeRequest:
					assert.IsType(t, VectorizeRequest{}, payload)
				case PipelineVectorizeCompleted:
					assert.IsType(t, VectorizeCompleted{}, payload)
				case PipelineFailed:
					assert.IsType(t, Failed{}, payload)
				case SagaStateChanged:
					assert.IsType(t, StateChanged{}, payload)
				}
			}
		})
	}
}

func TestKafkaConsumer_ValidateMessage(t *testing.T) {
	consumer := &KafkaConsumer{}

	tests := []struct {
		name        string
		envelope    Envelope[any]
		expectValid bool
	}{
		{
			name: "valid envelope",
			envelope: Envelope[any]{
				MessageID:  "msg-123",
				SagaID:     "saga-123",
				Type:       PipelineExtractRequest,
				OccurredAt: time.Now().UTC(),
				Payload:    "test payload",
				Meta: Meta{
					AppID:         "test-app",
					Initiator:     InitiatorSystem,
					Retries:       0,
					SchemaVersion: SchemaVersionV1,
				},
			},
			expectValid: true,
		},
		{
			name: "missing saga_id",
			envelope: Envelope[any]{
				MessageID:  "msg-123",
				Type:       PipelineExtractRequest,
				OccurredAt: time.Now().UTC(),
				Payload:    "test payload",
				Meta: Meta{
					AppID:         "test-app",
					Initiator:     InitiatorSystem,
					Retries:       0,
					SchemaVersion: SchemaVersionV1,
				},
			},
			expectValid: false,
		},
		{
			name: "missing type",
			envelope: Envelope[any]{
				MessageID:  "msg-123",
				SagaID:     "saga-123",
				OccurredAt: time.Now().UTC(),
				Payload:    "test payload",
				Meta: Meta{
					AppID:         "test-app",
					Initiator:     InitiatorSystem,
					Retries:       0,
					SchemaVersion: SchemaVersionV1,
				},
			},
			expectValid: false,
		},
		{
			name: "missing occurred_at",
			envelope: Envelope[any]{
				MessageID: "msg-123",
				SagaID:    "saga-123",
				Type:      PipelineExtractRequest,
				Payload:   "test payload",
				Meta: Meta{
					AppID:         "test-app",
					Initiator:     InitiatorSystem,
					Retries:       0,
					SchemaVersion: SchemaVersionV1,
				},
			},
			expectValid: false,
		},
		{
			name: "missing meta.app_id",
			envelope: Envelope[any]{
				MessageID:  "msg-123",
				SagaID:     "saga-123",
				Type:       PipelineExtractRequest,
				OccurredAt: time.Now().UTC(),
				Payload:    "test payload",
				Meta: Meta{
					Initiator:     InitiatorSystem,
					Retries:       0,
					SchemaVersion: SchemaVersionV1,
				},
			},
			expectValid: false,
		},
		{
			name: "missing meta.initiator",
			envelope: Envelope[any]{
				MessageID:  "msg-123",
				SagaID:     "saga-123",
				Type:       PipelineExtractRequest,
				OccurredAt: time.Now().UTC(),
				Payload:    "test payload",
				Meta: Meta{
					AppID:         "test-app",
					Retries:       0,
					SchemaVersion: SchemaVersionV1,
				},
			},
			expectValid: false,
		},
		{
			name: "missing meta.schema_version",
			envelope: Envelope[any]{
				MessageID:  "msg-123",
				SagaID:     "saga-123",
				Type:       PipelineExtractRequest,
				OccurredAt: time.Now().UTC(),
				Payload:    "test payload",
				Meta: Meta{
					AppID:     "test-app",
					Initiator: InitiatorSystem,
					Retries:   0,
				},
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := mustMarshal(tt.envelope)
			result, err := consumer.ValidateMessage(data)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectValid, result.Valid)

			if !tt.expectValid {
				assert.NotEmpty(t, result.Errors)
			}
		})
	}
}

func TestKafkaConsumer_LogMessageInfo(t *testing.T) {
	consumer := &KafkaConsumer{}

	// This test just ensures the method doesn't panic
	consumer.LogMessageInfo("test-saga", "test.event", "test payload")
}

func TestBuildEnvelopeWithMeta(t *testing.T) {
	payload := ExtractRequest{
		AppID:     "test-app",
		AppName:   "Test App",
		Countries: []string{"US"},
		DateFrom:  "2024-01-01",
		DateTo:    "2024-01-31",
	}

	envelope := BuildEnvelopeWithMeta(payload, PipelineExtractRequest, "test-saga", "custom-app", InitiatorUser)

	assert.NotEmpty(t, envelope.MessageID)
	assert.Equal(t, "test-saga", envelope.SagaID)
	assert.Equal(t, PipelineExtractRequest, envelope.Type)
	assert.False(t, envelope.OccurredAt.IsZero())
	assert.Equal(t, payload, envelope.Payload)
	assert.Equal(t, "custom-app", envelope.Meta.AppID)
	assert.Equal(t, InitiatorUser, envelope.Meta.Initiator)
	assert.Equal(t, SchemaVersionV1, envelope.Meta.SchemaVersion)
}

// Helper function to marshal JSON without error handling for tests
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
