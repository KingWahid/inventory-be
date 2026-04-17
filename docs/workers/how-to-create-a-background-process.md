# How to Create a Background Process

A step-by-step guide for adding a new Asynq-based background process to the workers service.

## Table of Contents

- [Overview](#overview)
- [How Background Processes Work](#how-background-processes-work)
- [Step 1: Add Type Constant](#step-1-add-type-constant)
- [Step 2: Create Task Handler](#step-2-create-task-handler)
- [Step 3: Register Handler](#step-3-register-handler)
- [Step 4: Create Enqueue Call](#step-4-create-enqueue-call)
- [Step 5: Test](#step-5-test)
- [Real-World Example: Invoice PDF](#real-world-example-invoice-pdf)
- [Checklist](#checklist)
- [Related Guides](#related-guides)

---

## Overview

Background processes are **user-triggered async tasks** — PDF generation, report creation, bulk data exports. They differ from scheduled jobs (which are time-based and system-internal).

**Use a background process when:**
- Work is triggered by a user action or API call
- The user needs to see status/progress
- The work produces a downloadable artifact (PDF, CSV, etc.)
- You need retry with backoff

**Key packages:**
- `pkg/common/background_processes/` — service, repository, enqueue (lifecycle management)
- `pkg/database/constants/background_process_type.go` — process types and Asynq task type mapping
- `workers/asynq/` — Asynq server, task handler dispatch
- `workers/tasks/` — per-type handler implementations

---

## How Background Processes Work

Before implementing a new process type, it helps to understand the architecture that makes background processes work.

### Two-phase architecture

A background process involves two separate systems working together:

**Phase 1 (service layer, synchronous):** When a user triggers a process (e.g., "generate invoice PDF"), the service creates a record in `common.background_processes` with `pending` status and enqueues a Redis task via Asynq. This happens in the API request — the response is fast because it only creates a record and enqueues, it doesn't do the actual work.

**Phase 2 (worker, asynchronous):** The Asynq worker picks up the task from Redis and runs the handler. The handler does the heavy lifting — fetching data, generating files, uploading to storage — and updates the process record as it goes.

### Why two separate systems?

The database record and the Asynq task serve different audiences. The **database record** exists so users can track status via the API (`GET /background-processes/{id}/status`). It stores status, progress percentage, and output (like a download URL). The **Asynq task** exists so the worker infrastructure knows what to process — it handles queuing, retry, and delivery. They're linked by the process ID stored in the task payload as `job_id`.

If enqueue to Redis fails after the database record is created, the service marks the record as `failed` with the enqueue error — so the user sees a clear failure instead of a perpetually-pending process.

### Status lifecycle

```
pending → processing → completed
                    → failed
                    → cancelled (user-initiated)
```

- **`pending`** — record created, task enqueued, waiting for a worker to pick it up. The user sees "queued."
- **`processing`** — worker picked it up and is actively working. Progress updates (0-100%) flow to the user.
- **`completed`** — work finished successfully. The output JSONB column contains results (e.g., download URL, file ID).
- **`failed`** — unrecoverable error occurred. The error message is stored for debugging.
- **`cancelled`** — user requested cancellation before or during processing.

### Terminal state guard

Once a process reaches `completed`, `failed`, or `cancelled`, **no further status updates are accepted**. The repository's `UpdateIfMutable()` method enforces this with a WHERE clause: `WHERE id = ? AND status IN ('pending', 'processing')`. If the status already changed to a terminal state, the UPDATE matches zero rows and returns an error.

This prevents race conditions. Consider: a user cancels a process, but the worker is in the middle of completing it. Without the guard, the worker could overwrite `cancelled` with `completed`, confusing the user. With the guard, the worker's `Complete()` call fails (zero rows affected), and the process stays cancelled.

---

## Step 1: Add Type Constant

Edit `pkg/database/constants/background_process_type.go` to register the new process type.

This step requires four changes because the type system has four responsibilities:

- **The type constant** (`BackgroundProcessTypeMyNewType`) — what the service layer uses to create the process record.
- **The Asynq task type** (`TaskTypeMyNewType`) — what the worker uses to route tasks to the correct handler. The `background_process:` prefix namespaces all background process tasks in the Asynq queue.
- **The `supportedBackgroundProcessTypes` map** — validates that the type is known before creating a record. Without this, a typo in the service layer would create an orphaned record with no handler.
- **The `TaskTypeFromProcessType` function** — maps between the two. Called by the enqueue service to determine which Asynq task type to use for a given process type.

### Add the type constant

```go
const (
    BackgroundProcessTypeInvoicePDF  = "invoice_pdf"
    BackgroundProcessTypeReport      = "report"
    BackgroundProcessTypeBulkExport  = "bulk_export"
    BackgroundProcessTypeMyNewType   = "my_new_type"  // <-- add here
)
```

### Add the Asynq task type string

```go
const (
    TaskTypeInvoicePDF  = "background_process:invoice_pdf"
    TaskTypeReport      = "background_process:report"
    TaskTypeBulkExport  = "background_process:bulk_export"
    TaskTypeMyNewType   = "background_process:my_new_type"  // <-- add here
)
```

### Update supportedBackgroundProcessTypes

```go
var supportedBackgroundProcessTypes = map[string]bool{
    BackgroundProcessTypeInvoicePDF:  true,
    BackgroundProcessTypeReport:      true,
    BackgroundProcessTypeBulkExport:  true,
    BackgroundProcessTypeMyNewType:   true,  // <-- add here
}
```

### Update TaskTypeFromProcessType

```go
func TaskTypeFromProcessType(processType string) string {
    switch processType {
    case BackgroundProcessTypeInvoicePDF:
        return TaskTypeInvoicePDF
    case BackgroundProcessTypeReport:
        return TaskTypeReport
    case BackgroundProcessTypeBulkExport:
        return TaskTypeBulkExport
    case BackgroundProcessTypeMyNewType:
        return TaskTypeMyNewType  // <-- add here
    default:
        return ""
    }
}
```

---

## Step 2: Create Task Handler

Create a new handler file in `workers/tasks/`. The handler runs inside the Asynq worker process.

**Reference:** `workers/tasks/invoice_pdf.go`

### Handler structure

```go
package tasks

import (
    "context"
    "encoding/json"

    "github.com/hibiken/asynq"
    "go.uber.org/zap"

    bgprocess "github.com/industrix-id/backend/pkg/common/background_processes"
    "github.com/industrix-id/backend/pkg/common/storage"
)

// myNewTypePayload matches the JSON payload from the enqueue call.
type myNewTypePayload struct {
    JobID string          `json:"job_id"`
    Input json.RawMessage `json:"input"`
}

// myNewTypeInput is the domain-specific input for this task.
type myNewTypeInput struct {
    OrganizationID string `json:"organization_id"`
    // ... task-specific fields ...
}

// HandleMyNewTypeTask processes a my_new_type background process.
func HandleMyNewTypeTask(
    ctx context.Context,
    t *asynq.Task,
    jobService bgprocess.BackgroundProcessService,
    storageClient storage.Client,
) error {
    // 1. Parse payload
    var payload myNewTypePayload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return err
    }

    var input myNewTypeInput
    if err := json.Unmarshal(payload.Input, &input); err != nil {
        return err
    }

    jobID := payload.JobID

    // 2. Mark as processing (10%)
    if err := jobService.UpdateStatus(ctx, jobID, "processing", 10); err != nil {
        return err
    }

    // 3. Check for early cancellation
    process, err := jobService.Get(ctx, jobID)
    if err != nil {
        return err
    }
    if process.Status == "cancelled" {
        return nil  // Return nil to suppress Asynq retry
    }

    // 4. Do the work (update progress as you go)
    // ... fetch data (30%) ...
    if err := jobService.UpdateStatus(ctx, jobID, "processing", 30); err != nil {
        return err
    }

    // ... process data (60%) ...
    if err := jobService.UpdateStatus(ctx, jobID, "processing", 60); err != nil {
        return err
    }

    // ... upload result to storage if needed (80%) ...
    if err := jobService.UpdateStatus(ctx, jobID, "processing", 80); err != nil {
        return err
    }

    // 5. Complete
    output := map[string]interface{}{
        "result_key": "result_value",
    }
    outputBytes, _ := json.Marshal(output)
    if err := jobService.Complete(ctx, jobID, outputBytes); err != nil {
        return err
    }

    return nil
}
```

### Retryable vs non-retryable errors

The most important decision in a background process handler is **which errors should trigger a retry and which should not**. The rule is simple: return the `error` if Asynq should retry, return `nil` if it should not.

**Return `nil` (suppress retry):** When the error is permanent — the data is missing, the input is invalid, or a business rule is violated. These won't fix themselves on retry. Call `jobService.Fail()` first to record the failure reason, then return `nil` so Asynq doesn't waste retries.

**Return `error` (allow Asynq retry):** When the error is transient — a network timeout, a temporary database lock, a storage service blip. Asynq will retry up to `MaxRetry` times (default 3) with exponential backoff, and the transient issue will likely resolve.

| Scenario | Action | Why |
|----------|--------|-----|
| Invoice not found in database | `Fail()` + return `nil` | Data issue won't fix itself on retry |
| PDF template rendering error | `Fail()` + return `nil` | Code/template bug, retry won't help |
| Storage upload timeout | return `error` | Transient network issue, retry likely succeeds |
| Database deadlock on file insert | return `error` | Deadlock is transient, retry likely succeeds |
| Invalid UUID in payload | `Fail()` + return `nil` | Malformed input, permanent |
| Redis connection refused | return `error` | Transient infra issue |

### Why progress tracking matters

Users see a progress bar in the UI. Without progress updates, the bar stays at 0% until completion — the user thinks it's stuck and might cancel a perfectly healthy process. Update at meaningful milestones (data fetched, file generated, file uploaded) so the UI reflects actual progress. Choose milestones that represent real work boundaries, not arbitrary percentages.

### Early cancellation check

Between enqueue and worker pickup, the user might cancel the process via the API. The worker checks the status after marking as `processing`. If it's already `cancelled`, the worker returns `nil` (no error, no retry). This avoids wasting resources on work the user no longer wants. The check happens early — before any heavy computation or external calls — to minimize wasted work.

### Key patterns

1. **Payload structure** — always `{"job_id": "...", "input": {...}}`. The `job_id` links back to the `background_processes` record.
2. **Progress tracking** — call `jobService.UpdateStatus(ctx, jobID, "processing", percent)` at meaningful milestones (10%, 30%, 60%, 80%, 100%).
3. **Storage integration** — use `storageClient.Upload()` for file outputs, then `storageClient.GetSignedDownloadURL()` for the download link stored in the process output.

---

## Step 3: Register Handler

Edit `workers/asynq/handlers.go` to register the new task type.

### Add to Handlers struct (if new dependencies needed)

```go
type Handlers struct {
    JobService    bgprocess.BackgroundProcessService
    StorageClient storage.Client
    // ... add new dependencies here if needed ...
}
```

### Add handler method

```go
func (h *Handlers) handleMyNewType(ctx context.Context, t *asynq.Task) error {
    return tasks.HandleMyNewTypeTask(ctx, t, h.JobService, h.StorageClient)
}
```

### Register in RegisterHandlers

```go
func RegisterHandlers(mux *asynq.ServeMux, h *Handlers) {
    mux.HandleFunc(constants.TaskTypeInvoicePDF, h.handleInvoicePDF)
    mux.HandleFunc(constants.TaskTypeReport, h.handleReport)
    mux.HandleFunc(constants.TaskTypeBulkExport, h.handleBulkExport)
    mux.HandleFunc(constants.TaskTypeMyNewType, h.handleMyNewType)  // <-- add here
}
```

---

## Step 4: Create Enqueue Call

In the service layer, use `BackgroundProcessService.Create()` to create the process record and enqueue the Asynq task.

```go
// In your service (e.g., pkg/services/myservice/service.go)

func (s *service) StartMyNewTypeProcess(ctx context.Context, orgID uuid.UUID, params MyParams) (*bgprocess.Process, error) {
    // Build the input payload
    input := map[string]interface{}{
        "organization_id": orgID.String(),
        // ... other fields ...
    }
    inputBytes, err := json.Marshal(input)
    if err != nil {
        return nil, common.NewCustomError("failed to marshal input").
            WithErrorCode(errorcodes.InternalError).
            WithHTTPCode(http.StatusInternalServerError)
    }

    // Create the background process (this also enqueues the Asynq task)
    process, err := s.bgProcessService.Create(ctx, bgprocess.CreateProcessParams{
        Type:           constants.BackgroundProcessTypeMyNewType,
        OrganizationID: orgID,
        UserID:         claims.UserID,  // from auth context
        Input:          inputBytes,
    })
    if err != nil {
        return nil, err
    }

    return process, nil
}
```

The `Create` method handles both phases:
1. Validating the process type against `supportedBackgroundProcessTypes`
2. Creating the `background_processes` database record with `pending` status
3. Enqueuing the Asynq task with the job ID and input
4. On enqueue failure, best-effort marking the process as `failed` with the enqueue error

### Default timeouts per type

The enqueue service applies default timeouts if not specified:
- `invoice_pdf`: 120 seconds
- `report`: 300 seconds
- `bulk_export`: 600 seconds

For a new type, add a default timeout in `pkg/common/background_processes/enqueue.go`.

---

## Step 5: Test

### Unit testing the task handler

```go
func TestHandleMyNewTypeTask(t *testing.T) {
    // Create mocks
    mockJobService := bgmocks.NewBackgroundProcessService(t)
    mockStorage := storagemocks.NewClient(t)

    // Set up expectations
    mockJobService.On("UpdateStatus", mock.Anything, "job-123", "processing", 10).Return(nil)
    mockJobService.On("Get", mock.Anything, "job-123").Return(&bgprocess.Process{Status: "processing"}, nil)
    mockJobService.On("UpdateStatus", mock.Anything, "job-123", "processing", 30).Return(nil)
    // ... more expectations ...
    mockJobService.On("Complete", mock.Anything, "job-123", mock.Anything).Return(nil)

    // Create Asynq task
    payload, _ := json.Marshal(map[string]interface{}{
        "job_id": "job-123",
        "input":  json.RawMessage(`{"organization_id": "org-456"}`),
    })
    task := asynq.NewTask("background_process:my_new_type", payload)

    // Run handler
    err := tasks.HandleMyNewTypeTask(context.Background(), task, mockJobService, mockStorage)
    require.NoError(t, err)

    mockJobService.AssertExpectations(t)
}
```

---

## Real-World Example: Invoice PDF

To see how all the pieces fit together, let's trace the `invoice_pdf` handler from enqueue to completion.

### The pipeline

The invoice PDF handler (`workers/tasks/invoice_pdf.go`) follows a 7-stage pipeline:

| Stage | Progress | What happens | Error handling |
|-------|----------|-------------|----------------|
| 1 | 10% | Mark as `processing` | Return error (retryable) |
| 2 | 30% | Fetch invoice with items (`GetByIDUnscoped`) | **Non-retryable**: invoice not found → `Fail()` + return `nil` |
| 3 | 40% | Fetch organization details for PDF branding | Warning only — continues without org if fetch fails |
| 4 | 60% | Generate PDF via `pdfService.GenerateInvoicePDF()` | Return error (retryable) |
| 5 | 80% | Upload PDF to S3 storage | Return error (retryable — transient network issues) |
| 6 | 90% | Create file registry record in `files` table | Return error (retryable — possible deadlock) |
| 7 | 100% | Generate signed download URL (1h TTL), mark as `completed` | Return error (retryable) |

### Why each stage exists

**Stage 2 (fetch invoice):** Uses `GetByIDUnscoped` because the invoice might have been soft-deleted between enqueue and processing. If the invoice doesn't exist, the handler calls `Fail()` and returns `nil` — there's nothing to retry.

**Stage 3 (fetch organization):** Org details are used for PDF branding (logo, company name). If the org fetch fails, the handler continues with a warning — a PDF without branding is better than no PDF at all.

**Stage 5 (upload to storage):** Uses `storageClient.Upload()` with the path `invoices/{invoice_id}/{invoice_number}.pdf`. Returns the error on failure so Asynq retries — storage timeouts are transient.

**Stage 6 (file registry):** Creates a record in the `files` table linking the storage path to the invoice. This lets the system track what files exist and clean them up later if needed.

**Stage 7 (signed URL):** Generates a 1-hour presigned download URL stored in the process output. The user fetches this URL via `GET /background-processes/{id}/status` and uses it to download the PDF.

### The output

On success, the process output looks like:

```json
{
    "download_url": "http://localhost:9000/documents/invoices/abc-123/INV-2024-001.pdf?X-Amz-...",
    "file_id": "def-456"
}
```

The user polls the status endpoint, sees `completed` with this output, and the frontend opens the download URL.

---

## Checklist

- [ ] Type constant added to `pkg/database/constants/background_process_type.go`
- [ ] Asynq task type string added
- [ ] `supportedBackgroundProcessTypes` map updated
- [ ] `TaskTypeFromProcessType` switch updated
- [ ] Task handler created in `workers/tasks/`
- [ ] Handler registered in `workers/asynq/handlers.go`
- [ ] Enqueue call created in service layer via `BackgroundProcessService.Create()`
- [ ] Default timeout added in `pkg/common/background_processes/enqueue.go` (if needed)
- [ ] Unit tests written and passing
- [ ] Build passes: `go build ./workers/...`

---

## Related Guides

- [How to Create a Scheduled Job](./how-to-create-a-scheduled-job.md) — for cron-based recurring jobs
- [How to Use Events](../service/how-to-use-events.md) — for event-driven communication
- [How to Use Storage](../service/how-to-use-storage.md) — for file upload/download in background processes
- [Workers README](../../workers/README.md) — workers service overview
