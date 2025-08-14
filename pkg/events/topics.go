package events

// Kafka topic constants for all ClientPulse services.
// These topics are used consistently across all services to ensure
// proper event routing and processing.
const (
	// Pipeline events
	PipelineExtractRequest   = "pipeline.extract_reviews.request"
	PipelineExtractCompleted = "pipeline.extract_reviews.completed"
	PipelinePrepareRequest   = "pipeline.prepare_reviews.request"
	PipelinePrepareCompleted = "pipeline.prepare_reviews.completed"
	PipelineFailed           = "pipeline.failed"

	// Saga orchestration events
	SagaStateChanged = "saga.orchestrator.state.changed"
)
