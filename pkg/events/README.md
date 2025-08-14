# Events Package

The `events` package provides shared Kafka event contracts and helpers used by all Quiby services.

## Features

- Standardized Event Envelope
- Type-Safe Payloads
- Built-in Validation
- Kafka Headers Support
- Schema Evolution

## Quick Start

```go
import "github.com/quiby-ai/common/pkg/events"

// Create metadata
meta := events.NewMeta("review-service", "tenant-123", events.InitiatorUser)

// Create payload
payload := events.ExtractRequest{
    AppID:     "com.example.app",
    AppName:   "Example App",
    Countries: []string{"US", "GB", "DE"},
    DateFrom:  "2025-01-01",
    DateTo:    "2025-01-31",
}

// Create envelope
envelope := events.NewEnvelope("saga-456", events.PipelineExtractRequest, payload, meta)

// Marshal to JSON
data, err := events.MarshalEnvelope(envelope)
```

## Event Types

- Pipeline events (extract, prepare, failed)
- Saga orchestration events
- All events use consistent envelope structure

## Validation

```go
result := events.ValidateEnvelope(envelope)
if !result.Valid {
    for _, err := range result.Errors {
        log.Printf("Field %s: %s", err.Field, err.Message)
    }
}
```
