# Events Package

This package provides a robust Kafka consumer and producer implementation with comprehensive payload validation and type safety for the ClientPulse review processing pipeline.

## Features

- **Type-safe payload handling**: Automatic payload type detection and validation based on event type
- **Comprehensive validation**: Built-in validation for all payload types using go-playground/validator
- **Structured logging**: Detailed logging for debugging and monitoring
- **Error handling**: Graceful error handling with specific error messages
- **Envelope format**: Standardized message envelope with metadata
- **Saga support**: Built-in support for saga orchestration patterns

## Architecture

### Message Flow

```
Producer → Envelope[T] → JSON → Kafka Topic → Consumer → Validated Payload → Processor
```

### Envelope Structure

```json
{
  "message_id": "uuid",
  "saga_id": "saga-uuid",
  "type": "pipeline.extract_reviews.request",
  "occurred_at": "2024-01-01T00:00:00Z",
  "payload": { ... },
  "meta": {
    "app_id": "review-ingestor",
    "initiator": "system",
    "retries": 0,
    "schema_version": "v1"
  }
}
```

## Usage

### Producer

```go
import "github.com/quiby-ai/common/pkg/events"

// Create producer
producer := events.NewKafkaProducer([]string{"localhost:9092"})
defer producer.Close()

// Create payload
extractReq := events.ExtractRequest{
    AppID:     "com.example.app",
    AppName:   "Example App",
    Countries: []string{"US", "GB"},
    DateFrom:  "2024-01-01",
    DateTo:    "2024-01-31",
}

// Validate payload
if err := extractReq.Validate(); err != nil {
    return err
}

// Build envelope
envelope := events.BuildEnvelope(extractReq, events.PipelineExtractRequest, "saga-123")

// Publish event
err := producer.PublishEvent(ctx, []byte("saga-123"), envelope)
```

### Consumer

```go
import "github.com/quiby-ai/common/pkg/events"

// Create consumer
consumer := events.NewTypedKafkaConsumer(
    []string{"localhost:9092"},
    "pipeline.extract_reviews.request",
    "review-ingestor-group",
)
defer consumer.Close()

// Create processor
processor := &MyProcessor{}
consumer.SetProcessor(processor)

// Start consuming
go func() {
    if err := consumer.Run(ctx); err != nil {
        log.Printf("Consumer error: %v", err)
    }
}()
```

### Processor Implementation

```go
type MyProcessor struct{}

func (p *MyProcessor) Handle(ctx context.Context, payload any, sagaID string) error {
    // Type switch to handle different payload types
    switch req := payload.(type) {
    case events.ExtractRequest:
        return p.handleExtractRequest(ctx, req, sagaID)
    case events.ExtractCompleted:
        return p.handleExtractCompleted(ctx, req, sagaID)
    // ... handle other types
    default:
        return fmt.Errorf("unknown payload type: %T", payload)
    }
}

func (p *MyProcessor) handleExtractRequest(ctx context.Context, req events.ExtractRequest, sagaID string) error {
    log.Printf("Processing extract request for app %s in saga %s", req.AppName, sagaID)
    
    // Validate the request
    if err := req.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // Process the request...
    return nil
}
```

## Event Types

### Pipeline Events

- `pipeline.extract_reviews.request` - ExtractRequest
- `pipeline.extract_reviews.completed` - ExtractCompleted
- `pipeline.prepare_reviews.request` - PrepareRequest
- `pipeline.prepare_reviews.completed` - PrepareCompleted
- `pipeline.vectorize_reviews.request` - VectorizeRequest
- `pipeline.vectorize_reviews.completed` - VectorizeCompleted
- `pipeline.failed` - Failed

### Saga Events

- `saga.orchestrator.state.changed` - StateChanged

## Payload Validation

All payload types implement the `Validate()` method using go-playground/validator:

```go
type ExtractRequest struct {
    AppID     string   `json:"app_id" validate:"required"`
    AppName   string   `json:"app_name" validate:"required"`
    Countries []string `json:"countries" validate:"required,min=1,dive,len=2"`
    DateFrom  string   `json:"date_from" validate:"required,datetime=2006-01-02"`
    DateTo    string   `json:"date_to" validate:"required,datetime=2006-01-02"`
}
```

## Error Handling

The consumer provides detailed error messages for common issues:

- **Invalid message format**: JSON parsing errors
- **Missing required fields**: saga_id, type, payload
- **Payload validation failures**: Field validation errors
- **Unknown event types**: Unsupported event types
- **Type mismatches**: Payload type doesn't match event type

## Logging

The consumer logs detailed information for debugging:

```
Processing message - SagaID: saga-123, Type: pipeline.extract_reviews.request, Payload: {AppID:com.example.app AppName:Example App Countries:[US GB] DateFrom:2024-01-01 DateTo:2024-01-31}
```

## Testing

Run the test suite:

```bash
go test ./pkg/events/... -v
```

The tests cover:
- Payload extraction and validation
- Message envelope validation
- Producer envelope building
- Consumer error handling
- Type safety verification

## Migration from Previous Version

### Before (Simple Structure)
```go
var fullMessage struct {
    SagaID  string `json:"saga_id"`
    Payload any    `json:"payload"`
}
```

### After (Envelope Structure)
```go
// Consumer automatically handles envelope format
// Producer creates proper envelopes with metadata
envelope := BuildEnvelope(payload, eventType, sagaID)
```

## Best Practices

1. **Always validate payloads** before processing
2. **Use type switches** to handle different payload types safely
3. **Set appropriate app_id and initiator** in meta fields
4. **Handle errors gracefully** and log relevant information
5. **Use structured logging** for better observability
6. **Test payload validation** with various input scenarios

## Troubleshooting

### Common Issues

1. **"invalid_payload_type" error**: Check that the event type matches the expected payload structure
2. **Validation failures**: Ensure all required fields are present and valid
3. **Missing meta fields**: Use `BuildEnvelope` or `BuildEnvelopeWithMeta` to ensure all required fields are set
4. **Type assertion panics**: Always use type switches to safely handle different payload types

### Debug Mode

Enable detailed logging by setting the log level to DEBUG in your application configuration.

## Contributing

When adding new event types:

1. Define the payload structure in `payloads.go`
2. Add validation tags and implement `Validate()` method
3. Add the event type constant in `topics.go`
4. Update the consumer's `extractAndValidatePayload` method
5. Add comprehensive tests
6. Update this documentation
