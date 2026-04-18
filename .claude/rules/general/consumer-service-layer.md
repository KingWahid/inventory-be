# Consumer Service Layer Pattern

## The Rule

**All consumer business logic must live in the service layer.** Consumer handlers in `workers/jobs/consumers/` must be thin — they unmarshal the event payload, delegate to a `ConsumerService` method, and return.

## Why

The same principle as HTTP handlers: transport is thin, business logic lives in services. This enables:

- **Unit testing without Redis infrastructure** — mock the ConsumerService interface
- **Reuse** — multiple consumers or scheduled jobs can call the same service method
- **Single responsibility** — the consumer handler handles event wiring, the service handles business logic
- **Consistent error handling** — services use `common.CustomError`, consumers handle retry/DLQ

## Pattern

### Service Layer (`pkg/services/<domain>/`)

Define a `ConsumerService` interface separate from the main `Service`:

```go
// consumer_service.go
type ConsumerService interface {
    HandleSomeEvent(ctx context.Context, payload SomeEventPayload) (*SomeResult, error)
}

type consumerService struct {
    // Only consumer-specific dependencies
    someRepo   some.Repository
    outboxRepo outbox_events.Repository
}

var _ ConsumerService = (*consumerService)(nil)
```

**Why a separate type?** The consumer may need different dependencies than the main service (e.g., `organizations.Repository` for recipient resolution that the main service doesn't need). Keeping them separate avoids bloating the main service struct.

### FX Module (`pkg/services/<domain>/consumer_module.go`)

```go
type ConsumerServiceParams struct {
    fx.In
    // consumer-specific dependencies
}

type ConsumerServiceResult struct {
    fx.Out
    ConsumerService ConsumerService
}

var ConsumerModule = fx.Module("<domain>_consumer",
    fx.Provide(ProvideConsumerService),
)
```

### Consumer Handler (`workers/jobs/consumers/<domain>/`)

```go
func (m *Manager) handleSomeEvent(ctx context.Context, msg *consumer.MessageContext) error {
    var payload service.SomeEventPayload
    if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
        return unmarshalError(err)
    }
    _, err := m.consumerService.HandleSomeEvent(ctx, payload)
    return err
}
```

The handler does three things only:
1. Unmarshal the payload
2. Delegate to the ConsumerService
3. Return the error

No business logic. No repository calls. No notification building. No audit logging.

## File Organization

Consumer service files follow the naming convention:

```
pkg/services/<domain>/
├── consumer_service.go          # Interface + struct + payload types
├── consumer_module.go           # FX wiring (ConsumerModule)
├── consumer_recipients.go       # Recipient resolution helpers (if needed)
├── consumer_notification.go     # Notification helpers (if needed)
├── consumer_handle_<event>.go   # One file per handler method
└── consumer_service_test.go     # Tests for all handler methods
```

## Examples

### Billing ConsumerService (canonical)

- Interface: `pkg/services/billing/consumer_service.go`
- Module: `pkg/services/billing/consumer_module.go`
- Handlers: `consumer_handle_invoice_sent.go`, `consumer_handle_refund_completed.go`, etc.
- Consumer wiring: `workers/jobs/consumers/billing_notification/`

### Feature ConsumerService

- Interface: `pkg/services/feature/consumer.go`
- Module: `pkg/services/feature/consumer_module.go`
- Consumer wiring: `workers/jobs/consumers/feature/`

## Forbidden

- **No repository calls in consumer handlers** — delegate to the ConsumerService
- **No outbox event creation in consumer handlers** — the service builds and enqueues notifications
- **No audit logging in consumer handlers** — the service handles audit entries
- **No template data building in consumer handlers** — the service constructs email/in-app payloads
