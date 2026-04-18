package usecase

// Event type strings written to outbox_events.event_type.
// Payload JSON shapes are documented in ARCHITECTURE.md §10 (Outbox event payloads table).
const (
	EventTypeMovementCreated     = "MovementCreated"
	EventTypeStockChanged        = "StockChanged"
	EventTypeStockBelowThreshold = "StockBelowThreshold"
)
