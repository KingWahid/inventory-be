# How to Create a Scheduled Job

A step-by-step guide for adding a new cron-based scheduled job to the workers service.

## Table of Contents

- [Overview](#overview)
- [How Scheduled Jobs Work](#how-scheduled-jobs-work)
- [Step 1: Choose a Handler Pattern](#step-1-choose-a-handler-pattern)
- [Step 2: Create the Handler File](#step-2-create-the-handler-file)
- [Step 3: Register the Handler](#step-3-register-the-handler)
- [Step 4: Seed the Job](#step-4-seed-the-job)
- [Step 5: Test](#step-5-test)
- [Real-World Example: Invoice Overdue Check](#real-world-example-invoice-overdue-check)
- [Checklist](#checklist)
- [Related Guides](#related-guides)

---

## Overview

Scheduled jobs are time-based periodic tasks that run on a cron schedule — stale process cleanup, auto-invoicing, quota initialization, outbox cleanup. They differ from background processes (which are user-triggered, Asynq-based, with progress tracking).

**Use a scheduled job when:**
- Work runs on a recurring schedule regardless of user actions
- No user needs to see the result
- The work is system maintenance (cleanup, auto-generation, health checks)
- Configuration should be changeable without redeployment (DB-driven cron expressions)

**Key packages:**
- `pkg/scheduler/` — core scheduler engine, distributed locker, handler registry
- `workers/jobs/scheduled/` — MODULE.go (wires all handler dependencies) + domain subdirectories
- `pkg/database/repositories/scheduled_jobs/` — repository for job definitions and run records

---

## How Scheduled Jobs Work

Before diving into implementation, it helps to understand the four systems that make scheduled jobs work.

### The scheduler loop

The scheduler (`pkg/scheduler/scheduler.go`) runs a reload loop every 60 seconds. Each cycle, it queries `common.scheduled_jobs` for all enabled jobs and reconciles them with the in-process gocron scheduler. If a job was added, its cron expression is registered. If a job was disabled or its schedule changed, gocron is updated accordingly. This means you can change a job's cron expression or config in the database and the scheduler picks it up within a minute — no redeployment needed.

When gocron determines a job's cron expression matches the current time, it calls the scheduler's `executeHandler` function. This function creates a context with the job's configured timeout and calls `registry.Execute(handlerName, config)`, which looks up the handler function by name and invokes it.

### Distributed locking

In production, multiple worker instances run simultaneously. Without coordination, every instance would execute the same job at the same time. The scheduler uses a **database-level distributed lock** to prevent this.

When gocron fires a job, it calls `PostgresLocker.Lock()` before executing the handler. The locker creates a run record in `common.scheduled_job_runs` with a `locked_by` column set to the current worker's instance ID. The key trick is a **UNIQUE constraint on `(job_id, scheduled_at)`** — `scheduled_at` is truncated to the minute, so only one worker per minute per job can insert a run record. The `INSERT ... ON CONFLICT` ensures this is atomic: the first worker to insert wins the lock, and all others skip the job.

After the handler completes, `Unlock()` persists the result (status, output, error message, completion timestamp) back to the run record.

### Run records

Every execution — successful or not — creates a row in `common.scheduled_job_runs`. Each row records:
- **Status**: `running`, `completed`, or `failed`
- **Timestamps**: `started_at`, `completed_at`, duration
- **Result**: A JSONB column with structured output from the handler (e.g., "10 invoices marked overdue, 2 errors")
- **Error**: The error message if the run failed
- **Locked by**: Which worker instance ran it

This is operational history for debugging and monitoring, not user-facing data. When something goes wrong ("why weren't invoices marked overdue last night?"), you query the run records to find out.

The scheduler also detects **stale runs** — runs that have been in `running` status longer than the job's timeout plus a 5-minute grace period. These are automatically marked as `failed`, freeing the lock for the next scheduled trigger.

### Context and config

The scheduler creates a context with a timeout derived from `timeout_seconds` in the job definition. If your handler runs longer than this, the context is cancelled. The handler should check `ctx.Done()` in any loops to respect this timeout.

The `config` column (JSONB) from the job definition is passed to the handler as `json.RawMessage`. This lets you tune job behavior (batch sizes, thresholds, feature flags) by updating the database row — no code change or redeployment required. For example, you could lower the batch size of a cleanup job during peak hours by updating its config.

---

## Step 1: Choose a Handler Pattern

There are two handler patterns. Choose based on your handler's dependency needs.

### Closure Pattern (recommended for new handlers)

Use when the handler needs **injected dependencies** (repositories, transaction manager, etc.). The closure captures dependencies at registration time and returns a handler function.

**Signature:** `func(ctx context.Context, config json.RawMessage) (json.RawMessage, error)`

**Reference:** `workers/jobs/scheduled/billing/invoice_overdue.go`

```go
// NewInvoiceOverdueCheckHandler returns a handler that marks invoices as overdue.
func NewInvoiceOverdueCheckHandler(
    db *gorm.DB,
    outboxRepo outbox_events.Repository,
    txManager transaction.Manager,
) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
    return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
        // Parse config with defaults
        cfg := invoiceOverdueConfig{BatchSize: commonconstants.DefaultScheduledJobBatchSize}
        if config != nil {
            if err := json.Unmarshal(config, &cfg); err != nil {
                logger.Warnw("Failed to parse config; using defaults", "error", err)
            }
        }

        // ... handler logic using db, outboxRepo, txManager ...

        return marshalResult(result), nil
    }
}
```

Each dependency serves a specific purpose:

- **`db *gorm.DB`** — needed when the handler queries or updates database tables directly (e.g., finding overdue invoices, marking processes as stale). The handler uses `db.WithContext(ctx)` to respect the scheduler's timeout — if the context is cancelled, in-flight queries are interrupted.
- **`outboxRepo outbox_events.Repository`** — needed when the handler must emit domain events. The outbox pattern guarantees the event is eventually published to Redis Streams even if Redis is temporarily down. Without this, you'd risk losing events during transient failures.
- **`txManager transaction.Manager`** — needed when the handler updates a record AND emits an event atomically. For example, the invoice overdue handler updates the invoice status and creates the outbox event in a single transaction — if either fails, both roll back. Without `txManager`, you could mark an invoice overdue but fail to emit the notification event, leaving the system in an inconsistent state.

Not every handler needs all three. A simple cleanup handler might only need `db`. A handler that emits events without modifying data might only need `outboxRepo`. Add only what the handler actually uses.

### SetDefaultDependencies Pattern

Use for **simpler handlers** where a package-level init is acceptable. Dependencies are wired once at startup via a setter function.

**Signature:** `func(ctx context.Context, config json.RawMessage) error`

**Reference:** `workers/jobs/scheduled/system/stale_background_process_cleanup.go`

```go
// Package-level dependencies, wired at startup.
type handlerDependencies struct {
    backgroundProcessRepo bgprocess.BackgroundProcessRepository
    scheduledRepo         scheduledrepo.Repository
}

var defaultDeps *handlerDependencies

var ErrHandlerNotInitialized = errors.New("handler not initialized")

func SetDefaultDependencies(bgRepo bgprocess.BackgroundProcessRepository, schedRepo scheduledrepo.Repository) {
    defaultDeps = &handlerDependencies{
        backgroundProcessRepo: bgRepo,
        scheduledRepo:         schedRepo,
    }
}

func StaleBackgroundProcessCleanup(ctx context.Context, _ json.RawMessage) error {
    if defaultDeps == nil {
        return ErrHandlerNotInitialized
    }
    // ... handler logic using defaultDeps.backgroundProcessRepo ...
    return nil
}
```

### Which Pattern to Choose?

| Criteria | Closure | SetDefaultDependencies |
|----------|---------|----------------------|
| Returns structured result? | Yes — `(json.RawMessage, error)` | No — `error` only |
| Dependencies injected via | Function parameters | Package-level setter |
| Testability | Easier (pass mock deps directly) | Requires calling setter in test setup |
| Existing examples | `invoice_overdue`, `outbox_cleanup`, `ftm_daily_quota_init` | `stale_background_process_cleanup`, `pending_upload_cleanup`, `invoice_auto_generation` |

**Why `(json.RawMessage, error)` vs `error`?** The result signature lets the scheduler store structured output in `scheduled_job_runs.result`. This is invaluable for debugging — "how many invoices were marked overdue? were there errors? which ones failed?" Without a result, you only get pass/fail, and debugging requires digging through logs.

**Default to the closure pattern** for new handlers unless you have a specific reason not to.

---

## Step 2: Create the Handler File

Create a new file in the appropriate domain subdirectory under `workers/jobs/scheduled/`.

### Directory placement

| Domain | Directory | Examples |
|--------|-----------|----------|
| Billing | `workers/jobs/scheduled/billing/` | invoice overdue, auto-generation |
| System maintenance | `workers/jobs/scheduled/system/` | stale process cleanup |
| Common/uploads | `workers/jobs/scheduled/common/` | pending upload cleanup |
| FTM | `workers/jobs/scheduled/ftm/` | daily quota initialization |
| Outbox | `workers/jobs/scheduled/outbox/` | outbox cleanup |
| New domain | `workers/jobs/scheduled/{domain}/` | Create a new subdirectory |

### Handler structure (closure pattern)

```go
package billing

import (
    "context"
    "encoding/json"

    "go.uber.org/zap"
    "gorm.io/gorm"
)

// Config struct — define fields with sensible defaults.
type myJobConfig struct {
    BatchSize int `json:"batch_size"`
}

// Result struct — recorded in scheduled_job_runs.result (JSONB).
type myJobResult struct {
    TotalProcessed int      `json:"total_processed"`
    TotalErrors    int      `json:"total_errors"`
    Errors         []string `json:"errors,omitempty"`
}

// NewMyJobHandler returns a handler for the my_job scheduled job.
func NewMyJobHandler(db *gorm.DB) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
    return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
        // 1. Parse config with defaults
        cfg := myJobConfig{BatchSize: 100}
        if config != nil {
            if err := json.Unmarshal(config, &cfg); err != nil {
                logger.Warnw("Failed to parse my_job config; using defaults", "error", err)
            }
        }
        if cfg.BatchSize <= 0 {
            cfg.BatchSize = 100
        }

        result := myJobResult{}

        logger.Infow("Starting my_job", "batch_size", cfg.BatchSize)

        // 2. Process in batches with context cancellation check
        for {
            select {
            case <-ctx.Done():
                return marshalMyJobResult(result), ctx.Err()
            default:
            }

            // ... query and process a batch ...
            // result.TotalProcessed += len(batch)

            // Break when no more items
            break
        }

        logger.Infow("my_job completed",
            "total_processed", result.TotalProcessed,
            "total_errors", result.TotalErrors,
        )

        return marshalMyJobResult(result), nil
    }
}

func marshalMyJobResult(r myJobResult) json.RawMessage {
    b, err := json.Marshal(r)
    if err != nil {
        return nil
    }
    return b
}
```

### Key design decisions explained

**Why config parsing is lenient:** `json.Unmarshal` errors are logged as warnings, not returned as errors. A config typo shouldn't prevent the job from running — it falls back to safe defaults and continues. The operator can fix the config in the database and the corrected version takes effect on the next cron trigger. If config parsing returned an error, a typo would fail every run until someone deploys a fix or manually edits the database.

**Why context cancellation matters:** The scheduler enforces `timeout_seconds` via the context. If your handler runs longer than the configured timeout, the context is cancelled. Without the `select { case <-ctx.Done() }` check in batch loops, the handler would continue running after the scheduler considers it timed out. This creates two problems: the stale-run detector might mark it failed while it's still running, and the next scheduled trigger might start a new run that overlaps with the stale one.

**Why per-item error tolerance:** A job processing 1000 items should not fail entirely because item #3 had bad data. Instead, collect errors in the result struct, continue processing remaining items, and report the summary. The operator can investigate individual failures from the run's `result` JSON. This is especially important for cleanup and maintenance jobs — one corrupted record shouldn't prevent cleaning up the other 999.

**Why result marshaling returns nil on error:** If marshaling the result struct fails (unlikely but possible), the handler returns `nil` instead of an error. The job's actual work is already done — failing the entire run because of a serialization issue in the result would be misleading and could trigger unnecessary retries.

---

## Step 3: Register the Handler

Edit `workers/jobs/scheduled/MODULE.go` to register your handler.

MODULE.go is where FX provides all the dependencies that handlers need. The closure pattern works here because `RegisterHandlers` receives the FX-injected `Params` struct at startup, instantiates each closure with the dependencies it needs, and registers the resulting function with the handler registry. When the scheduler later triggers "my_job", the registry looks up the function by name and calls it with `(ctx, config)`. The dependencies are already captured inside the closure — no further injection needed at runtime.

### For closure-pattern handlers (with result)

```go
// In RegisterHandlers function:

// Wire and register my_job (returns result)
myJobHandler := billing.NewMyJobHandler(p.DB)
p.Registry.RegisterWithResult("my_job", myJobHandler)
```

### For SetDefaultDependencies handlers (error only)

```go
// In RegisterHandlers function:

// Wire dependencies
mypackage.SetDefaultDependencies(p.SomeRepo)

// Register handler
p.Registry.Register("my_job", mypackage.MyJobHandler)
```

### Add dependencies to Params

If your handler needs a dependency not already in the `Params` struct, add it:

```go
type Params struct {
    fx.In

    DB                    *gorm.DB
    BackgroundProcessRepo bgprocess.BackgroundProcessRepository
    ScheduledRepo         scheduledrepo.Repository
    OutboxRepo            outbox_events.Repository
    TransactionManager    transaction.Manager
    HealthServer          *health.Server
    Registry              *scheduler.HandlerRegistry
    MyNewRepo             mynewrepo.Repository  // <-- add here
}
```

### Include sub-package FX modules

If your handler uses the `SetDefaultDependencies` pattern with its own `module.go`, include its `Module` in the parent module:

```go
var Module = fx.Module("scheduled_jobs",
    commonhandlers.Module,
    billing.Module,
    mynewpackage.Module,  // <-- add here
    fx.Invoke(RegisterHandlers),
)
```

**Current MODULE.go for reference:** `workers/jobs/scheduled/MODULE.go`

---

## Step 4: Seed the Job

Add a seed row to `infra/database/cmd/seed/crons/0003_scheduled_jobs.yaml` so the job exists in the database.

The seed file is the single source of truth for job definitions. It runs during deployment and local setup, ensuring every environment has the same set of jobs with consistent configuration.

### YAML format

```yaml
up:
  execute:
    # ... existing jobs ...

    # My Job - describe what it does
    - >-
      INSERT INTO common.scheduled_jobs
        (name, description, cron_expression, handler, config, enabled, timeout_seconds)
      VALUES
        ('my_job', 'Short description of what this job does', '0 4 * * *', 'my_job', '{"batch_size": 100}', true, 120)
      ON CONFLICT (name)
      DO UPDATE SET
        description = EXCLUDED.description,
        cron_expression = EXCLUDED.cron_expression,
        handler = EXCLUDED.handler,
        config = EXCLUDED.config,
        enabled = EXCLUDED.enabled,
        timeout_seconds = EXCLUDED.timeout_seconds;

down:
  execute:
    # ... existing cleanup ...
    - DELETE FROM common.scheduled_job_runs WHERE job_id IN (SELECT id FROM common.scheduled_jobs WHERE name = 'my_job');
    - DELETE FROM common.scheduled_jobs WHERE name = 'my_job';
```

**Why `ON CONFLICT (name) DO UPDATE`?** Seeds must be idempotent — running them multiple times shouldn't create duplicates or fail. The `ON CONFLICT` clause means: if a job with this name already exists, update its config/schedule instead of erroring. This is critical for deployments where seeds run automatically on every deploy. It also means you can change a job's schedule or config by updating the seed file and redeploying — the existing row gets updated in place.

### Field descriptions

| Field | Description | Example |
|-------|-------------|---------|
| `name` | Unique job identifier. Must match the handler name in `Registry.Register()` | `'my_job'` |
| `description` | Human-readable description for debugging | `'Cleans up stale records'` |
| `cron_expression` | Standard cron expression (minute, hour, day-of-month, month, day-of-week) | `'0 4 * * *'` (4 AM daily) |
| `handler` | Handler name registered in the scheduler registry | `'my_job'` |
| `config` | JSON config passed to the handler, or `NULL` if no config needed | `'{"batch_size": 100}'` |
| `enabled` | Whether the job is active. Set `false` for jobs not yet implemented | `true` |
| `timeout_seconds` | Max execution time before the scheduler marks the run as stale | `120` |

### Common cron expressions

| Schedule | Expression |
|----------|------------|
| Every 5 minutes | `*/5 * * * *` |
| Every hour at :00 | `0 * * * *` |
| Daily at midnight UTC | `0 0 * * *` |
| Daily at 2 AM UTC | `0 2 * * *` |
| Weekly on Monday at 3 AM | `0 3 * * 1` |

### Apply the seed

```bash
make -C infra/database seed-crons
```

---

## Step 5: Test

### Unit testing a closure-pattern handler

```go
func TestMyJobHandler(t *testing.T) {
    // Create mock dependencies
    db := setupTestDB(t)

    // Create handler
    handler := NewMyJobHandler(db)

    // Run with config
    config := json.RawMessage(`{"batch_size": 10}`)
    result, err := handler(context.Background(), config)

    require.NoError(t, err)

    var res myJobResult
    require.NoError(t, json.Unmarshal(result, &res))
    assert.Equal(t, 0, res.TotalErrors)
}
```

### Unit testing a SetDefaultDependencies handler

```go
func TestStaleCleanup(t *testing.T) {
    // Wire mock dependencies
    mockBgRepo := bgmocks.NewBackgroundProcessRepository(t)
    mockSchedRepo := schedmocks.NewRepository(t)
    SetDefaultDependencies(mockBgRepo, mockSchedRepo)

    mockBgRepo.On("MarkStaleProcessesAsFailed", mock.Anything).Return(nil, nil)
    mockSchedRepo.On("MarkStaleRunsAsFailed", mock.Anything).Return(nil, nil)

    err := StaleBackgroundProcessCleanup(context.Background(), nil)
    require.NoError(t, err)
}
```

---

## Real-World Example: Invoice Overdue Check

To see how all the pieces fit together, let's trace the `invoice_overdue_check` job from trigger to completion.

### What triggers it

The seed file defines: `cron_expression: '0 0 * * *'` — midnight UTC daily. The scheduler's reload loop picks this up, registers it with gocron, and gocron fires the handler at midnight.

### How the handler works

1. **Lock acquisition**: The `PostgresLocker` creates a run record with `scheduled_at` truncated to the minute. If another worker already claimed this minute's run, the INSERT fails on the unique constraint and this worker skips the job.

2. **Config parsing**: The handler reads `{"batch_size": 100}` from the job's config column. If parsing fails, it falls back to the default batch size.

3. **Batch query**: The handler queries invoices in batches:
   ```sql
   WHERE status IN ('sent', 'unpaid', 'partially_paid')
     AND due_date < NOW()
     AND deleted_at IS NULL
   LIMIT 100
   ```

4. **Per-invoice processing**: Each overdue invoice is processed in its own transaction via `txManager.RunInTx()`:
   - **Guarded UPDATE**: Sets `status = 'overdue'` with a WHERE clause that rechecks the status (`WHERE status IN ('sent', 'unpaid', 'partially_paid')`). If another process already changed the status, `RowsAffected == 0` and the invoice is silently skipped — no error, no retry.
   - **Outbox event**: In the same transaction, creates an outbox event with type `billing.invoice.overdue` containing invoice details and `days_overdue`. This event eventually triggers a notification to the customer.
   - **Atomicity**: Both the status update and the outbox event commit or roll back together.

5. **Error isolation**: If processing one invoice fails (e.g., a database error), the error is logged and added to the result's `Errors` array. Processing continues with the remaining invoices.

6. **Result**: The handler returns a structured result:
   ```json
   {
     "total_marked": 47,
     "total_errors": 2,
     "errors": ["invoice abc-123: foreign key violation", "invoice def-456: timeout"]
   }
   ```
   This is stored in `scheduled_job_runs.result`, giving operators full visibility into what happened.

7. **Unlock**: The `PostgresLocker` persists the result and marks the run as `completed` (or `failed` if the handler returned an error).

### What makes this a good reference

- **Batch processing** with context cancellation checks between batches
- **Per-item isolation** — one failure doesn't abort the whole job
- **Transactional consistency** — status update + outbox event in one transaction
- **Defensive updates** — WHERE clause prevents race conditions
- **Structured results** — operator visibility into counts and errors

---

## Checklist

- [ ] Handler file created in `workers/jobs/scheduled/{domain}/`
- [ ] Handler follows one of the two patterns (closure or SetDefaultDependencies)
- [ ] Config struct has sensible defaults
- [ ] Context cancellation checked in batch loops
- [ ] Handler registered in `workers/jobs/scheduled/MODULE.go`
- [ ] New dependencies added to `Params` struct (if needed)
- [ ] Sub-package FX module included in parent `Module` (if using SetDefaultDependencies)
- [ ] Seed row added to `infra/database/cmd/seed/crons/0003_scheduled_jobs.yaml`
- [ ] Seed applied: `make -C infra/database seed-crons`
- [ ] Unit tests written and passing
- [ ] Build passes: `go build ./workers/...`

---

## Related Guides

- [How to Create a Background Process](./how-to-create-a-background-process.md) — for user-triggered async tasks with progress tracking
- [How to Use Events](../service/how-to-use-events.md) — for event-driven communication between services
- [Workers README](../../workers/README.md) — workers service overview
