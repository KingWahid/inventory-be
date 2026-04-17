# How to Use the Event System

A comprehensive guide to the Redis Streams event bus architecture for inter-service communication.

## Table of Contents

- [Overview](#overview)
- [Architecture Flow](#architecture-flow)
- [Key Concepts](#key-concepts)
  - [Events](#events)
  - [Outbox Pattern](#outbox-pattern)
  - [Consumer Groups](#consumer-groups)
  - [Idempotency](#idempotency)
- [Naming Conventions](#naming-conventions)
- [Publishing Events](#publishing-events)
- [Consuming Events](#consuming-events)
  - [Directory Structure](#directory-structure)
  - [Creating a Consumer](#creating-a-consumer)
  - [Handler Implementation](#handler-implementation)
  - [Middleware Chain](#middleware-chain)
  - [ConsumerService Pattern](#consumerservice-pattern)
- [Defining New Events](#defining-new-events)
- [Configuration](#configuration)
- [Error Handling & DLQ](#error-handling--dlq)
- [Security](#security)
- [Testing](#testing)
- [Related Guides](#related-guides)

---

## Overview

The event system enables asynchronous, decoupled communication between services using Redis Streams. It provides:

- **Transactional Guarantees**: Events are written to the database in the same transaction as domain changes (Outbox Pattern)
- **At-Least-Once Delivery**: Consumer groups ensure messages are processed at least once
- **Idempotency**: Built-in deduplication prevents duplicate processing
- **Dead Letter Queue (DLQ)**: Failed messages are moved to a DLQ after max retries
- **HMAC Signing**: Events are cryptographically signed for integrity verification

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Architecture Flow

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   Service A      │     │  PostgreSQL      │     │ Workers Service  │     │  Redis Streams   │
│   (Producer)     │     │  (outbox_events) │     │  (Outbox Worker) │     │                  │
│                  │     │                  │     │                  │     │                  │
│  1. Domain Logic │────▶│ 2. Insert Event  │────▶│ 3. Poll/Publish  │────▶│ 4. XADD message  │
│     + Outbox TX  │     │    (same TX)     │     │    + HMAC Sign   │     │                  │
└──────────────────┘     └──────────────────┘     └──────────────────┘     └────────┬─────────┘
                                                                                     │
                                                                                     ▼
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│   Consumer       │     │  Consumer        │     │  Redis           │     │  Consumer Group  │
│   (Workers)      │◀────│  Handler         │◀────│  Idempotency     │◀────│  XREADGROUP      │
│   Action Done!   │     │  Thin Delegate   │     │  Check & Mark    │     │  5. Read message │
└──────────────────┘     └──────────────────┘     └──────────────────┘     └──────────────────┘
```

**Flow Summary:**
1. **Service A** performs domain logic and writes an outbox event in the same transaction
2. **Outbox table** stores the event with payload, ensuring atomic commit
3. **Workers service** polls the outbox (with LISTEN/NOTIFY optimization) and publishes to Redis
4. **Redis Streams** stores the event with HMAC signature
5. **Consumer Group** (XREADGROUP) delivers messages to consumers in the workers service
6. **Consumer handler** checks idempotency, delegates to ConsumerService, and acknowledges

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Key Concepts

### Events

Events are immutable facts that something happened in the system. Each event has:

```go
// BaseEvent is the envelope for all events
type BaseEvent struct {
    ID          string    `json:"id"`           // UUID - unique identifier
    Type        string    `json:"type"`         // e.g., "notification.email.requested"
    Version     int       `json:"version"`      // Schema version (for evolution)
    Stream      string    `json:"stream"`       // Target Redis stream name
    CreatedAt   time.Time `json:"created_at"`   // When the event was created
    PublishedAt time.Time `json:"published_at"` // When published to Redis (for replay protection)
    Payload     []byte    `json:"payload"`      // JSON-encoded domain data
    Signature   string    `json:"signature"`    // HMAC-SHA256 for integrity
}
```

**Domain events** contain the actual business data:

```go
// Example: EmailRequested event payload
type EmailRequested struct {
    RequestID    uuid.UUID              `json:"request_id"`
    To           string                 `json:"to"`
    Locale       string                 `json:"locale"`
    TemplateName string                 `json:"template_name"`
    SubjectArgs  map[string]interface{} `json:"subject_args,omitempty"`
    BodyArgs     map[string]interface{} `json:"body_args,omitempty"`
    RequestedAt  time.Time              `json:"requested_at"`
}
```

### Outbox Pattern

**Why not publish directly to Redis?** If you publish to Redis directly from the service, and Redis is down (even briefly), the event is lost. The domain change commits to the database, but the event never reaches consumers — leading to inconsistent state. With the outbox pattern, the event is stored in the same database transaction as the domain change. Even if Redis is down for hours, the outbox publisher will pick up unpublished events when it recovers. The trade-off is slightly higher latency (the poll interval between insert and publish) in exchange for **zero event loss**.

The **Outbox Pattern** ensures events are never lost by writing them to the database in the same transaction as domain changes:

```go
// In your service layer - use txManager.RunInTx for transactions
func (s *service) CreateUser(ctx context.Context, user *User) error {
    return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // 1. Perform domain logic - repositories use transaction from context
        if err := s.userRepo.Create(txCtx, user); err != nil {
            return err
        }

        // 2. Create outbox event (same transaction!)
        event := &schemas.OutboxEvent{
            EventType:     "user.created",
            AggregateType: "user",
            AggregateID:   user.ID.String(),
            Payload:       json.RawMessage(`{"user_id": "..."}`),
            StreamName:    "user:profile:v1",
        }
        if err := s.outboxRepo.Create(txCtx, event); err != nil {
            return err
        }

        return nil  // Both committed together or both rolled back
    })
}
```

The **Workers service** then:
1. Polls the outbox table for unpublished events
2. Signs each event with HMAC-SHA256
3. Publishes to Redis Streams via XADD
4. Marks the event as published

### Consumer Groups

Redis Streams consumer groups provide:

- **Load balancing**: Messages are distributed among consumers
- **Tracking**: Redis tracks which messages each consumer has received
- **Acknowledgment**: Consumers must ACK messages after processing
- **Pending Entry List (PEL)**: Unacknowledged messages can be reclaimed

```
Stream: notification:email:v1
        ┌─────────────────────────────────────────────┐
        │ msg1 │ msg2 │ msg3 │ msg4 │ msg5 │ msg6 │...│
        └─────────────────────────────────────────────┘
                        ▲
                        │
Consumer Group: notification-email-consumer
        ┌──────────────────────────────────────┐
        │ Consumer 1 (pod-abc): msg2, msg4     │  ← receives subset
        │ Consumer 2 (pod-def): msg3, msg5     │  ← receives subset
        │ Consumer 3 (pod-ghi): msg1, msg6     │  ← receives subset
        └──────────────────────────────────────┘
```

### Idempotency

Idempotency ensures events are processed exactly once, even if delivered multiple times:

```go
// Redis-backed idempotency store
store := idempotency.NewRedisStore(
    redisClient,
    idempotency.WithTTL(24 * time.Hour),           // Dedup window
    idempotency.WithKeyPrefix("notification:idem"), // Key prefix
)

// Usage in middleware
isDuplicate, err := store.CheckAndMark(ctx, eventID)
if isDuplicate {
    return nil  // Already processed, skip
}

// Process the event...

// On failure, remove the mark to allow retry
if err := processEvent(ctx, msg); err != nil {
    store.Remove(ctx, eventID)
    return err
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Naming Conventions

### Stream Names

**Pattern**: `{domain}:{entity}:v{version}`

| Domain | Stream Examples |
|--------|-----------------|
| notification | `notification:email:v1` |
| ftm | `ftm:process:v1`, `ftm:device:v1` |
| user | `user:profile:v1`, `user:permission:v1` |
| billing | `billing:quota:v1` |

### Event Types

**Pattern**: `{domain}.{entity}.{action}`

| Examples |
|----------|
| `notification.email.requested` |
| `notification.email.sent` |
| `notification.email.failed` |
| `ftm.process.started` |
| `user.created` |

### Consumer Groups

**Pattern**: `{service}-{entity}-consumer`

| Examples |
|----------|
| `notification-email-consumer` |
| `billing-quota-consumer` |

### Consumer IDs

**Pattern**: `{service}-{hostname/pod-id}`

| Examples |
|----------|
| `notification-pod-abc123` |
| `billing-pod-def456` |

### Dead Letter Queue (DLQ)

**Pattern**: `dlq:{original_stream}`

| Examples |
|----------|
| `dlq:notification:email:v1` |
| `dlq:ftm:process:v1` |

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Publishing Events

Events are published through the **Outbox Pattern**. Write to the outbox table in your service:

```go
// In your service/business logic - call within txManager.RunInTx callback
func (s *service) SendPasswordResetEmail(txCtx context.Context, user *User) error {
    // 1. Create the domain event payload
    payload := notification.EmailRequested{
        RequestID:    uuid.New(),
        To:           user.Email,
        Locale:       user.PreferredLocale,
        TemplateName: "password_reset",
        BodyArgs: map[string]interface{}{
            "user_name":  user.Name,
            "reset_link": "https://example.com/reset?token=...",
        },
        RequestedAt: time.Now().UTC(),
    }

    // 2. Marshal to JSON
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal email payload: %w", err)
    }

    // 3. Insert into outbox (txCtx already contains transaction from caller)
    outboxEvent := &schemas.OutboxEvent{
        EventType:     notification.EmailRequestedType,  // "notification.email.requested"
        AggregateType: "user",
        AggregateID:   user.ID.String(),
        Payload:       payloadBytes,
        StreamName:    notification.EmailStreamName,     // "notification:email:v1"
    }

    return s.outboxRepo.Create(txCtx, outboxEvent)
}
```

**Important:** The Workers service handles the actual publishing to Redis. Your service only writes to the outbox table.

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Consuming Events

All event consumers live in the **workers service**, not in individual HTTP services.

### Directory Structure

```
workers/jobs/consumers/{domain}/
├── config.go        # Config struct embedding consumer.BaseConfig
├── manager.go       # Manager struct embedding consumer.BaseManager
├── MODULE.go        # FX module with lifecycle hooks (fx.Invoke)
├── handler_*.go     # Handler methods (thin: unmarshal → delegate to ConsumerService)
└── *_test.go        # Handler unit tests
```

**Canonical example:** `workers/jobs/consumers/notification/`

### Creating a Consumer

**Step 1: Create config** (`config.go`)

Embed `consumer.BaseConfig` and map fields from the workers config:

```go
package mydomainconsumer

import (
    "github.com/industrix-id/backend/pkg/eventbus/consumer"
    "github.com/industrix-id/backend/workers/config"
)

// Config holds consumer-specific configuration.
type Config struct {
    consumer.BaseConfig
}

// NewConfig creates a Config from the workers config.
func NewConfig(cfg *config.Config) *Config {
    return &Config{
        BaseConfig: consumer.BaseConfig{
            Enabled:        cfg.MyDomainConsumerEnabled,
            Group:          cfg.MyDomainConsumerGroup,
            MaxRetries:     cfg.OutboxMaxRetries,
            IdempotencyTTL: cfg.IdempotencyTTL,
            HMACSecret:     cfg.EventHMACSecret,
            RedisHost:      cfg.RedisHost,
            RedisPort:      cfg.RedisPort,
            RedisPassword:  cfg.RedisPassword,
            RedisStreamsDB:  cfg.RedisStreamsDB,
        },
    }
}

// Validate checks required fields.
func (c *Config) Validate() error {
    return c.BaseConfig.Validate()
}
```

**Step 2: Create manager** (`manager.go`)

Embed `consumer.BaseManager` and use `AddWorker()` with `DefaultMiddlewareChain()`:

```go
package mydomainconsumer

import (
    "go.uber.org/zap"

    "github.com/industrix-id/backend/pkg/eventbus/consumer"
    "github.com/industrix-id/backend/pkg/eventbus/events/mydomain"
    myservice "github.com/industrix-id/backend/pkg/services/mydomain"
)

var logger = zap.S().Named("consumers.mydomain")

// Manager embeds BaseManager for Redis Streams consumer lifecycle.
type Manager struct {
    *consumer.BaseManager
    consumerService myservice.ConsumerService
}

// NewManager creates a Manager with workers for each subscribed stream.
func NewManager(consumerService myservice.ConsumerService, cfg *Config) (*Manager, error) {
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    base, err := consumer.NewBaseManager("mydomain", &cfg.BaseConfig,
        consumer.WithIdempotencyKeyPrefix("mydomain:idempotency"),
    )
    if err != nil {
        return nil, err
    }

    m := &Manager{
        BaseManager:     base,
        consumerService: consumerService,
    }

    m.initWorkers()
    return m, nil
}

func (m *Manager) initWorkers() {
    m.createEntityWorker()
}

func (m *Manager) createEntityWorker() {
    m.AddWorker(
        mydomain.EntityStreamName,         // "mydomain:entity:v1"
        m.handleEntityEvent,               // handler method
        m.DefaultMiddlewareChain(),         // logging → signature → idempotency
    )
}
```

**Why BaseManager?** `BaseManager` handles everything about the Redis Streams lifecycle so you don't have to: creating the consumer group if it doesn't exist, reading messages with XREADGROUP, acknowledging processed messages, reclaiming unacknowledged messages from crashed consumers (via the Pending Entry List), and moving failed messages to DLQ after max retries. Without it, you'd write ~200 lines of Redis protocol code per consumer. By embedding it, your consumer only needs to define which streams to listen to and what to do with each message.

**Key points:**
- `AddWorker()` registers a worker for a specific stream with middleware
- `DefaultMiddlewareChain()` provides logging, HMAC signature validation, and idempotency — you don't need to build these manually
- For custom consumer groups (when multiple consumers read the same stream independently), pass `consumer.WithGroup(streams.MustBuildConsumerGroup("domain", "entity"))`

**Step 3: Create handler methods** (`handler_entity.go`)

Handlers are **thin** — unmarshal, delegate, return:

```go
package mydomainconsumer

import (
    "context"
    "encoding/json"

    "github.com/industrix-id/backend/pkg/eventbus/consumer"
    myservice "github.com/industrix-id/backend/pkg/services/mydomain"
)

func (m *Manager) handleEntityEvent(ctx context.Context, msg *consumer.MessageContext) error {
    var payload myservice.EntityEventPayload
    if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
        return unmarshalError(err)
    }

    _, err := m.consumerService.HandleEntityEvent(ctx, payload)
    return err
}
```

**Rules:**
- No business logic in handlers — delegate to the ConsumerService
- No repository calls in handlers
- No outbox event creation in handlers
- No notification building or audit logging in handlers

**Step 4: Create FX module** (`MODULE.go`)

Use `fx.Invoke` with lifecycle hooks for start/stop:

```go
package mydomainconsumer

import (
    "context"

    "go.uber.org/fx"
    "go.uber.org/zap"

    myservice "github.com/industrix-id/backend/pkg/services/mydomain"
    "github.com/industrix-id/backend/workers/config"
    "github.com/industrix-id/backend/workers/internal/health"
)

type Params struct {
    fx.In
    Lifecycle       fx.Lifecycle
    Config          *config.Config
    ConsumerService myservice.ConsumerService
    HealthServer    *health.Server
}

func RegisterLifecycle(p Params) {
    cfg := NewConfig(p.Config)

    if !cfg.Enabled {
        logger.Info("mydomain consumer disabled")
        return
    }

    manager, err := NewManager(p.ConsumerService, cfg)
    if err != nil {
        logger.With(zap.Error(err)).Error("Failed to create mydomain consumer manager")
        return
    }

    p.HealthServer.RegisterChecker("mydomain_consumer", manager.HealthCheck)

    p.Lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            logger.Info("Starting mydomain consumer")
            return manager.Start(ctx)
        },
        OnStop: func(ctx context.Context) error {
            logger.Info("Stopping mydomain consumer")
            return manager.Stop(ctx)
        },
    })
}

var Module = fx.Module("mydomain_consumer",
    fx.Invoke(RegisterLifecycle),
)
```

**Step 5: Import module** in `workers/cmd/main.go`

Add the module to the FX app composition:

```go
// Event consumers
notification.Module,
subscription.Module,
feature.Module,
mydomainconsumer.Module,  // <-- add here
```

### Handler Implementation

Handlers follow the thin-handler pattern. The handler's only responsibilities:

1. **Unmarshal** the event payload from `msg.Payload()` into a typed struct
2. **Delegate** to the `ConsumerService` method
3. **Return** the error (the middleware chain handles retry/DLQ)

```go
func (m *Manager) handleEmailRequested(ctx context.Context, msg *consumer.MessageContext) error {
    var payload notification.EmailRequested
    if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
        return unmarshalError(err)
    }

    err := m.notificationService.SendEmailNotification(
        payload.To,
        &payload.Locale,
        payload.TemplateName,
        toStringMap(payload.SubjectArgs),
        toStringMap(payload.BodyArgs),
    )
    return err
}
```

### Middleware Chain

The `DefaultMiddlewareChain()` provided by `BaseManager` applies three middleware in order. Each exists for a specific reason:

```
Request arrives
    ↓
┌─ Logging Middleware ─────────────────────────────┐
│  Log event_id, event_type, delivery_count        │
│      ↓                                           │
│  ┌─ Signature Middleware ─────────────────────┐  │
│  │  Verify HMAC-SHA256 signature              │  │
│  │  Reject if invalid                         │  │
│  │      ↓                                     │  │
│  │  ┌─ Idempotency Middleware ─────────────┐  │  │
│  │  │  Check if event_id already processed │  │  │
│  │  │  If duplicate → skip                 │  │  │
│  │  │  If new → process, mark on success   │  │  │
│  │  │      ↓                               │  │  │
│  │  │  ┌─ Handler ──────────────────────┐  │  │  │
│  │  │  │  Unmarshal → Delegate → Return │  │  │  │
│  │  │  └────────────────────────────────┘  │  │  │
│  │  └──────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

**Logging middleware (outermost):** Records `event_id`, `event_type`, stream, and `delivery_count` for every message, plus success/failure with duration. Essential for debugging — "did the event arrive? how many times was it retried? how long did processing take?" It's outermost so it captures all events, including those rejected by later middleware.

**Signature validation middleware:** Events are HMAC-signed by the outbox publisher using a shared secret. This middleware verifies the signature before processing. It also provides **replay protection**: events older than 5 minutes (`MaxAge`) are rejected, with 1 minute of clock skew tolerance. This prevents tampered or injected events from being processed — important because Redis Streams has no built-in authentication per message.

**Idempotency middleware (innermost):** At-least-once delivery means the same event may arrive multiple times — network retries, consumer crashes before ACK, pending message reclamation. The idempotency middleware checks a Redis key (`event_id`) before processing and marks it after success. If the event was already processed, it's silently skipped and ACK'd. If the handler fails, the idempotency mark is **removed** so the event can be retried. The TTL (default 24h) controls the deduplication window.

**The order matters:** Logging is outermost so it sees everything. Signature validation runs early to reject forged events before any processing. Idempotency is innermost — right before the handler — so deduplication happens after all safety checks pass.

For implementation details, see `pkg/eventbus/README.md`.

### ConsumerService Pattern

All consumer business logic must live in the **service layer**, not in consumer handlers. This keeps handlers thin and business logic testable without Redis infrastructure.

**Define a ConsumerService interface** in `pkg/services/{domain}/consumer_service.go`:

```go
type ConsumerService interface {
    HandleEntityCreated(ctx context.Context, payload EntityCreatedPayload) (*NotificationResult, error)
    HandleEntityUpdated(ctx context.Context, payload EntityUpdatedPayload) (*NotificationResult, error)
}

type consumerService struct {
    invoiceRepo   invoices.Repository
    orgRepo       organizations.Repository
    outboxRepo    outbox_events.Repository
    // ... consumer-specific dependencies
}

var _ ConsumerService = (*consumerService)(nil)
```

**Create a separate FX module** in `pkg/services/{domain}/consumer_module.go`:

```go
type ConsumerServiceParams struct {
    fx.In
    InvoiceRepo invoices.Repository
    OrgRepo     organizations.Repository
    OutboxRepo  outbox_events.Repository
}

type ConsumerServiceResult struct {
    fx.Out
    ConsumerService ConsumerService
}

var ConsumerModule = fx.Module("{domain}_consumer",
    fx.Provide(ProvideConsumerService),
)
```

**Why a separate type from the main Service?** Consumer handlers need different dependencies than HTTP handlers. For example, the billing `ConsumerService` needs `organizations.Repository` (to resolve notification recipients from an organization's users) and `outboxRepo` (to emit follow-up events like notifications), but doesn't need the main service's invoice CRUD methods or the billing calculation logic. Conversely, the main `Service` needs `invoiceRepo` for CRUD but doesn't need `organizations.Repository` for recipient resolution. Separate types keep each service focused on its specific concern and prevent dependency bloat — adding a consumer dependency doesn't force rebuilding or retesting the main service.

**Canonical reference:** `pkg/services/billing/consumer_service.go`

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Defining New Events

**Step 1: Create event definitions** in `pkg/eventbus/events/{domain}/{entity}.go`:

```go
package mydomain

import (
    "time"

    "github.com/google/uuid"

    "github.com/industrix-id/backend/pkg/eventbus/events"
)

// Stream name for this entity
const ProcessStreamName = "ftm:process:v1"

// Event type constants
const (
    ProcessStartedType = "ftm.process.started"
    ProcessStoppedType = "ftm.process.stopped"
)

// ProcessStarted is emitted when a process starts
type ProcessStarted struct {
    ProcessID   uuid.UUID `json:"process_id"`
    DeviceID    uuid.UUID `json:"device_id"`
    StartedAt   time.Time `json:"started_at"`
    StartedByID uuid.UUID `json:"started_by_id"`
}

func (e ProcessStarted) Type() string       { return ProcessStartedType }
func (e ProcessStarted) StreamName() string { return ProcessStreamName }

// Factory function for creating wrapped events
func NewProcessStartedEvent(payload ProcessStarted) (*events.BaseEvent, error) {
    return events.NewBaseEventFromPayload(ProcessStartedType, 1, ProcessStreamName, payload)
}

// Register event types for deserialization
func init() {
    events.Register(ProcessStartedType, func() interface{} { return &ProcessStarted{} })
    events.Register(ProcessStoppedType, func() interface{} { return &ProcessStopped{} })
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Configuration

Consumer configuration lives in the **workers service** config (`workers/config/config.go`). Each consumer domain has enable/group fields:

```go
// In workers/config/config.go
type Config struct {
    // ... existing fields ...

    // Notification consumer
    NotificationConsumerEnabled bool   `mapstructure:"NOTIFICATION_CONSUMER_ENABLED"`
    NotificationConsumerGroup   string `mapstructure:"NOTIFICATION_CONSUMER_GROUP"`

    // My domain consumer
    MyDomainConsumerEnabled bool   `mapstructure:"MY_DOMAIN_CONSUMER_ENABLED"`
    MyDomainConsumerGroup   string `mapstructure:"MY_DOMAIN_CONSUMER_GROUP"`

    // Shared consumer settings
    OutboxMaxRetries int    `mapstructure:"OUTBOX_MAX_RETRIES"`
    IdempotencyTTL   string `mapstructure:"IDEMPOTENCY_TTL"`
    EventHMACSecret  string `mapstructure:"EVENT_HMAC_SECRET"`
    RedisStreamsDB   int    `mapstructure:"REDIS_STREAMS_DB"`
}
```

### Environment Variables

```env
# Event Bus Security
EVENT_HMAC_SECRET=your-256-bit-secret-key-for-event-signing

# Redis Streams (separate DB from cache)
REDIS_STREAMS_DB=1

# Consumer settings
NOTIFICATION_CONSUMER_ENABLED=true
NOTIFICATION_CONSUMER_GROUP=notification-email-consumer
OUTBOX_MAX_RETRIES=3
IDEMPOTENCY_TTL=24h
```

### Docker Compose

Consumer environment variables are configured in `docker-compose.yaml` under the workers service:

```yaml
workers:
  environment:
    - NOTIFICATION_CONSUMER_ENABLED=true
    - NOTIFICATION_CONSUMER_GROUP=notification-email-consumer
    - OUTBOX_MAX_RETRIES=3
    - IDEMPOTENCY_TTL=24h
    - EVENT_HMAC_SECRET=${EVENT_HMAC_SECRET}
    - REDIS_STREAMS_DB=1
  depends_on:
    - industrix-redis
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Error Handling & DLQ

### Retry Logic

The consumer worker automatically retries failed messages:

1. Message fails processing
2. Worker checks retry count against `MaxRetries`
3. If under limit, message stays in PEL (Pending Entry List)
4. Pending recovery loop reclaims and retries after `ClaimMinIdle`
5. If max retries exceeded, message moves to DLQ

### Dead Letter Queue

Messages that fail after max retries are moved to a DLQ:

```
Original Stream: notification:email:v1
DLQ Stream:      dlq:notification:email:v1
```

DLQ messages include additional metadata:
- `original_stream`: The source stream
- `original_message_id`: The original Redis message ID
- `error`: The last error message
- `delivery_count`: Number of delivery attempts
- `moved_to_dlq_at`: Timestamp when moved to DLQ

### Handling DLQ Messages

DLQ processing can be done manually or via a scheduled job:

```go
// Example: Read and process DLQ messages
func (m *Manager) processDLQ(ctx context.Context) {
    dlqStream := streams.BuildDLQName("notification:email:v1")

    // Read messages from DLQ
    msgs, err := m.redisClient.XRange(ctx, dlqStream, "-", "+").Result()
    if err != nil {
        return
    }

    for _, msg := range msgs {
        // Analyze and potentially retry or escalate
        // After handling, delete from DLQ
        m.redisClient.XDel(ctx, dlqStream, msg.ID)
    }
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Security

### HMAC Signing

All events are signed with HMAC-SHA256 before publishing:

```go
// Signing input: eventID|publishedAt|payload
data := fmt.Sprintf("%s|%d|%s", eventID, publishedAt.UnixNano(), payload)
signature := hmac.New(sha256.New, secret).Sum([]byte(data))
```

### Signature Validation

The `DefaultMiddlewareChain()` includes automatic signature validation. If you need custom validation:

```go
func (m *Manager) signatureValidationMiddleware() consumer.HandlerMiddleware {
    return func(next consumer.MessageHandler) consumer.MessageHandler {
        return func(ctx context.Context, msg *consumer.MessageContext) error {
            if m.signer != nil {
                valid := m.signer.Verify(
                    msg.Event.ID,
                    msg.Event.PublishedAt,
                    msg.Event.Payload,
                    msg.Event.Signature,
                )
                if !valid {
                    return common.NewCustomError("invalid event signature").
                        WithErrorCode(errorcodes.InvalidSignature)
                }
            }
            return next(ctx, msg)
        }
    }
}
```

**Note:** The `DefaultMiddlewareChain()` already includes signature validation. Only implement custom middleware if you have special requirements.

### Replay Protection

The `PublishedAt` timestamp prevents replay attacks:
- Events older than 5 minutes are rejected
- Clock skew tolerance: 1 minute

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Testing

### Unit Testing Consumer Handlers

Consumer handlers are thin, so unit tests focus on verifying the unmarshal → delegate → return flow. Use a test struct that avoids needing a real Redis connection:

```go
//go:build !integration

package mydomainconsumer

import (
    "context"
    "encoding/json"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/industrix-id/backend/pkg/eventbus/consumer"
    "github.com/industrix-id/backend/pkg/eventbus/events"
)

// testManager avoids real Redis connection in unit tests.
type testManager struct {
    consumerService *mockConsumerService
}

func (tm *testManager) handleEntityEvent(ctx context.Context, msg *consumer.MessageContext) error {
    // Same logic as the real handler
    var payload EntityEventPayload
    if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
        return err
    }
    _, err := tm.consumerService.HandleEntityEvent(ctx, payload)
    return err
}

func TestHandleEntityEvent_Success(t *testing.T) {
    mockSvc := &mockConsumerService{}
    mockSvc.On("HandleEntityEvent", mock.Anything, mock.Anything).Return(nil, nil)

    tm := &testManager{consumerService: mockSvc}

    payload, _ := json.Marshal(EntityEventPayload{ID: "test-123"})
    msg := consumer.NewTestMessageContext(payload, "mydomain.entity.created")

    err := tm.handleEntityEvent(context.Background(), msg)

    assert.NoError(t, err)
    mockSvc.AssertExpectations(t)
}
```

**Reference:** `workers/jobs/consumers/notification/in_app_handler_test.go`

### Unit Testing ConsumerService

ConsumerService methods contain the business logic and are tested with standard mocks — no Redis infrastructure needed:

```go
func TestHandleEntityCreated(t *testing.T) {
    mockRepo := repomocks.NewRepository(t)
    mockOutbox := outboxmocks.NewRepository(t)

    svc := newConsumerService(mockRepo, mockOutbox)

    mockRepo.On("GetByID", mock.Anything, "entity-123").Return(&entity, nil)
    mockOutbox.On("Create", mock.Anything, mock.Anything).Return(nil)

    result, err := svc.HandleEntityCreated(ctx, EntityCreatedPayload{ID: "entity-123"})

    require.NoError(t, err)
    assert.NotNil(t, result)
    mockRepo.AssertExpectations(t)
}
```

### Test patterns

| What to test | Where | How |
|-------------|-------|-----|
| Handler unmarshal + delegation | `*_test.go` in consumer package | `testManager` struct with mock service |
| Business logic (recipients, notifications, audit) | `consumer_service_test.go` in service package | Standard mock repos |
| Invalid JSON handling | Consumer handler test | Verify error return on bad payload |
| Service error propagation | Consumer handler test | Mock service returns error, verify handler returns it |

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Create a Service](./how-to-create-a-service.md)
- [How to Write Service Layer](./how-to-write-service-layer.md)
- [How to Create a Scheduled Job](../workers/how-to-create-a-scheduled-job.md)
- [How to Create a Background Process](../workers/how-to-create-a-background-process.md)
- [Workers README](../../workers/README.md)
