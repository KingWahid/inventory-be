# Background Processes vs Scheduled Jobs

The workers service runs two distinct systems for background work. They share the same Go binary but serve different purposes and have different architectures. Do not conflate them.

## Background Processes

**What:** User-triggered async work — PDF generation, report creation, bulk data exports.

**Trigger:** A service call (e.g. billing service triggers invoice PDF). The service creates a `BackgroundProcess` record in `common.background_processes`, then enqueues it to Redis via Asynq.

**Status tracking:** `common.background_processes` table with status (`pending` → `processing` → `completed`/`failed`/`cancelled`), progress (0-100), and output (JSONB). Status transitions are guarded — once a process reaches a terminal state, it cannot be overwritten.

**User-facing API:** Yes. Users can list their processes, check status, and cancel pending/processing ones via the common service endpoints (`GET /background-processes`, `GET /background-processes/{id}/status`, `POST /background-processes/{id}/cancel`).

**Key packages:**
- `pkg/common/background_processes/` — service, repository, enqueue (lifecycle management)
- `pkg/database/repositories/background_processes/` — API-facing queries (list, get by ID, cancel)
- `workers/asynq/` — Asynq server, task handler dispatch
- `workers/tasks/` — per-type handler implementations (invoice_pdf, report, bulk_export)
- `pkg/database/constants/background_process_type.go` — process types and Asynq task type mapping

**Retry:** Asynq built-in retry (configurable per task, default `MaxRetry(3)`).

**Storage:** Background processes may upload/download files via the storage client (`pkg/common/storage/`).

## Scheduled Jobs

**What:** Time-based periodic system tasks — stale process cleanup, pending upload cleanup, auto-generation of invoices.

**Trigger:** Cron expressions. Job definitions are stored in `common.scheduled_jobs` (name, handler, cron expression, config, timeout, enabled flag). The scheduler reloads from the database every 60 seconds.

**Status tracking:** `common.scheduled_job_runs` table with per-run records (status, started_at, completed_at, locked_by, result, error). This is operational history for debugging, not user-facing.

**User-facing API:** No. Scheduled jobs are internal infrastructure. No endpoints expose them to users.

**Key packages:**
- `pkg/scheduler/` — core scheduler engine, distributed locker, handler registry
- `workers/jobs/scheduled/` — MODULE.go (wires all handler dependencies and registers them)
- `workers/jobs/scheduled/system/` — system handlers (stale process cleanup)
- `workers/jobs/scheduled/billing/` — billing handlers (invoice overdue check, auto-generation)
- `workers/jobs/scheduled/outbox/` — outbox cleanup handler
- `workers/jobs/scheduled/ftm/` — FTM daily quota initialization handler
- `workers/jobs/scheduled/common/` — common handlers (pending upload cleanup)
- `pkg/database/repositories/scheduled_jobs/` — repository for job definitions and run records

**Retry:** No built-in retry. If a run fails, it waits for the next cron trigger. The scheduler detects stale runs (running longer than timeout + 5min) and marks them failed.

**Concurrency control:** Distributed lock via database — first worker to claim a run wins (optimistic locking with `locked_by` column).

## Comparison

| Aspect | Background Processes | Scheduled Jobs |
|--------|---------------------|----------------|
| Trigger | Service call (user or system action) | Cron schedule (time-based) |
| Queue | Redis via Asynq | In-process cron scheduler (`go-co-op/gocron/v2`) |
| DB table | `common.background_processes` | `common.scheduled_jobs` + `common.scheduled_job_runs` |
| User-facing | Yes (list, status, cancel) | No (internal only) |
| Handler signature | `func(ctx, *asynq.Task) error` | `func(ctx, json.RawMessage) error` or `func(ctx, json.RawMessage) (json.RawMessage, error)` |
| Retry | Asynq retry (configurable) | Next cron trigger |
| Concurrency | Redis queue ordering | DB-level distributed lock |
| Storage integration | Yes (file upload/download) | No (typically) |
| Examples | Invoice PDF, reports, bulk exports | Stale process cleanup, auto-invoicing, invoice overdue check, outbox cleanup, FTM daily quota init |

## When to Use Which

**Use a background process when:**
- Work is triggered by a user action or API call
- The user needs to see status/progress
- The work produces a downloadable artifact (PDF, CSV, etc.)
- You need retry with backoff

**Use a scheduled job when:**
- Work runs on a recurring schedule regardless of user actions
- No user needs to see the result
- The work is system maintenance (cleanup, auto-generation, health checks)
- Configuration should be changeable without redeployment (DB-driven cron expressions)

## Adding a New Background Process Type

1. Add the type constant in `pkg/database/constants/background_process_type.go`
2. Add the Asynq task type mapping in the same file (`TaskTypeFromProcessType`)
3. Create the task handler in `workers/tasks/`
4. Register the handler in `workers/asynq/handlers.go` (`RegisterHandlers`)
5. Create the service method that calls `BackgroundProcessService.Create()` with the new type

## Adding a New Scheduled Job

1. Create the handler in a domain subdirectory under `workers/jobs/scheduled/` (e.g., `system/`, `billing/`, `outbox/`, `ftm/`, or a new one)
   - Use the **closure pattern** for handlers with dependencies: `NewXxxHandler(deps) func(ctx, json.RawMessage) (json.RawMessage, error)`
   - Use package-level `SetDefaultDependencies` only for simple handlers with no structured result
2. Register the handler in `workers/jobs/scheduled/MODULE.go` (`RegisterHandlers`):
   - `p.Registry.Register("handler_name", handlerFunc)` for simple handlers returning only `error`
   - `p.Registry.RegisterWithResult("handler_name", resultHandlerFunc)` for handlers returning `(json.RawMessage, error)`
3. Add any new dependencies to the `Params` struct in `workers/jobs/scheduled/MODULE.go`
4. Add a seed row to `infra/database/cmd/seed/crons/0003_scheduled_jobs.yaml` with the handler name, cron expression, config, and timeout
