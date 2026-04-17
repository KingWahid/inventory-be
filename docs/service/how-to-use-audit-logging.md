# How to Use Audit Logging

A guide to recording user actions, device operations, and system events for compliance and change tracking.

## Table of Contents

- [How to Use Audit Logging](#how-to-use-audit-logging)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Architecture](#architecture)
  - [Adding Audit Logging to a Service](#adding-audit-logging-to-a-service)
    - [1. Inject AuditLogger via FX](#1-inject-auditlogger-via-fx)
    - [2. Add auditlog.Module to main.go](#2-add-auditlogmodule-to-maingo)
    - [3. Call Log in Service Methods](#3-call-log-in-service-methods)
  - [Entry Fields](#entry-fields)
  - [Usage Patterns](#usage-patterns)
    - [User Action (HTTP)](#user-action-http)
    - [Device Action (MQTT/FTM)](#device-action-mqttftm)
    - [System Action (Scheduled Job)](#system-action-scheduled-job)
    - [Log Only After a Successful Transaction](#log-only-after-a-successful-transaction)
  - [Metadata](#metadata)
    - [When to Use Metadata](#when-to-use-metadata)
    - [Metadata Patterns by Domain](#metadata-patterns-by-domain)
    - [Metadata Rules](#metadata-rules)
  - [Constants](#constants)
    - [Features](#features)
    - [Entity Types](#entity-types)
    - [Actions](#actions)
    - [Client Types](#client-types)
  - [Sensitive Data Scrubbing](#sensitive-data-scrubbing)
  - [Middleware (IP and User Agent)](#middleware-ip-and-user-agent)
  - [Testing](#testing)
  - [What NOT to Do](#what-not-to-do)
  - [Package Structure](#package-structure)
  - [Related](#related)

---

## Overview

Audit logging records **what changed, who changed it, and when** for compliance and debugging. Every mutation in a service (create, update, delete, approve, reject, etc.) should produce an audit log entry.

Key properties:
- **Fire-and-forget** — `Log()` returns nothing. Audit failures are logged as warnings, never block the main operation.
- **Asynchronous** — entries are persisted in a background goroutine.
- **Auto-scrubbed** — sensitive keys in `OldValue`/`NewValue` are automatically redacted before storage.
- **One method** — `Log(ctx, Entry)`. No variants, no overloads.

## Architecture

```
Handler                        Service                          AuditLogger
───────                        ───────                          ───────────
                               s.auditLogger.Log(ctx, Entry{
Middleware sets IP+UA on ctx → OrganizationID, UserID,       → Reads IP+UA from ctx
(AuditContextMiddleware)       Feature, EntityType,             Scrubs sensitive keys
                               EntityID, Action,                Fires goroutine
                               OldValue, NewValue,                → repo.CreateLog
                               })
```

**Who provides what:**

| Data | Source | Mechanism |
|------|--------|-----------|
| OrganizationID (entity's owning org), UserID (for user actions) | Service method params; handlers pass claims / resource org | `Entry.OrganizationID`, `Entry.UserID` |
| DeviceID, DeviceUserID | Service method params (FTM) | `Entry.DeviceID`, `Entry.DeviceUserID` |
| IP Address | AuditContextMiddleware | `ctx` → `ExtractIPAddress(ctx)` |
| User Agent | AuditContextMiddleware | `ctx` → `ExtractUserAgent(ctx)` |
| Feature, EntityType, Action | Service method (knows the domain) | Entry fields |
| OldValue, NewValue | Service method (has the data) | Entry fields — auto-scrubbed |

## Adding Audit Logging to a Service

### 1. Inject AuditLogger via FX

In your service's `MODULE.go`, add `AuditLogger` to `ServiceParams`:

```go
import "github.com/industrix-id/backend/pkg/database/services/auditlog"

type ServiceParams struct {
    fx.In
    // ... existing dependencies ...
    AuditLogger auditlog.AuditLogger
}
```

In your service's `SERVICE.go`, accept it in the constructor and store it:

```go
type service struct {
    // ... existing fields ...
    auditLogger auditlog.AuditLogger
}

func newServiceWithDependencies(
    // ... existing params ...
    auditLogger auditlog.AuditLogger,
) *service {
    return &service{
        // ...
        auditLogger: auditLogger,
    }
}
```

### 2. Add auditlog.Module to main.go

In the service's `cmd/main.go`, add the module in the "Shared utility modules" section:

```go
import "github.com/industrix-id/backend/pkg/database/services/auditlog"

app := fx.New(
    // ...
    transaction.Module,
    auditlog.Module,  // after transaction.Module
    // ...
)
```

### 3. Call Log in Service Methods

After a successful mutation, call `Log`:

```go
s.auditLogger.Log(ctx, auditlog.Entry{
    OrganizationID: orgID,
    UserID:         &userID,
    Feature:        constants.FeatureBilling.Value,
    EntityType:     string(constants.EntityTypeBillingContact),
    EntityID:       contact.ID.String(),
    Action:         string(constants.ActionCreated),
    NewValue:       map[string]any{"id": contact.ID, "name": contact.Name},
})
```

That's it. No error handling needed — `Log` returns nothing.

## Entry Fields

```go
type Entry struct {
    OrganizationID uuid.UUID      // Required. Tenant scope.
    UserID         *uuid.UUID     // Who performed the action. Nil for system actions.
    DeviceID       *uuid.UUID     // IoT device (FTM). Nil for web actions.
    DeviceUserID   *uuid.UUID     // Device operator (FTM). Nil for web actions.
    Feature        string         // Module: "billing", "ftm", "access_control", etc.
    EntityType     string         // What was changed: "invoice", "payment_method", etc.
    EntityID       string         // ID of the changed entity.
    Action         string         // What happened: "created", "updated", "deleted", etc.
    ClientType     string         // "web", "edge_device", "system". Default: "" (inferred as web).
    OldValue       any            // State before change. map, struct, or nil. Auto-scrubbed.
    NewValue       any            // State after change. map, struct, or nil. Auto-scrubbed.
    Metadata       map[string]any // Optional extra context (e.g., flow_rate, reason).
}
```

**Rules:**
- `OrganizationID` — required. Use the **entity's owning organization** (e.g. the org the resource belongs to), not necessarily the request context org. When an admin acts on another org's resource (e.g. delete role), set `OrganizationID` to that resource's org so the audit is attributed to the correct tenant. See `pkg/services/access/delete_role.go` (targetOrgID from existingRole).
- `UserID` — required for user-initiated actions (HTTP, device). Pass `&userID` (pointer). For system actions (scheduled jobs, auto-generated), set `ClientType` to `"system"` and leave `UserID` nil. Handlers must pass the acting user (e.g. `claims.UserID`) into the service so the service can set `Entry.UserID`.
- `OldValue`/`NewValue` — pass `map[string]any` with the fields that matter. Don't dump entire structs.
- `Metadata` — contextual information about the action (email, platform_code, locale, geospatial data, operation type). See [Metadata](#metadata) section for patterns and rules. Not auto-scrubbed — never put sensitive data here.

## Usage Patterns

### User Action (HTTP)

The most common pattern. The **handler** must pass `userID` (e.g. `claims.UserID` from JWT) and `orgID` into the service; the **service** then sets `Entry.UserID: &userID` so the audit record identifies who performed the action.

```go
func (s *service) CreateBillingContact(ctx context.Context, orgID, userID uuid.UUID, req *types.CreateBillingContactRequest) (*types.BillingContactResponse, error) {
    // ... business logic, DB calls ...

    s.auditLogger.Log(ctx, auditlog.Entry{
        OrganizationID: orgID,
        UserID:         &userID,
        Feature:        constants.FeatureBilling.Value,
        EntityType:     string(constants.EntityTypeBillingContact),
        EntityID:       contact.ID.String(),
        Action:         string(constants.ActionCreated),
        NewValue:       map[string]any{"id": contact.ID, "user_id": contact.UserID, "is_active": contact.IsActive},
    })

    return response, nil
}
```

### Device Action (MQTT/FTM)

FTM operations come from edge devices. Pass device and user identifiers explicitly.

```go
s.auditLogger.Log(ctx, auditlog.Entry{
    OrganizationID: deviceUser.Device.OrganizationID,
    UserID:         &deviceUser.UserID,
    DeviceID:       &deviceUser.DeviceID,
    DeviceUserID:   &deviceUser.ID,
    Feature:        constants.FeatureFTM.Value,
    EntityType:     string(constants.EntityTypeProcess),
    EntityID:       processID.String(),
    Action:         string(constants.ActionStarted),
    ClientType:     string(constants.ClientTypeEdgeDevice),
    Metadata:       map[string]any{"flow_rate": flowRate},
})
```

### System Action (Scheduled Job)

No user involved. Set `ClientType` to `"system"` and leave `UserID` nil.

```go
s.auditLogger.Log(ctx, auditlog.Entry{
    OrganizationID: orgID,
    Feature:        constants.FeatureBilling.Value,
    EntityType:     string(constants.EntityTypeInvoice),
    EntityID:       invoice.ID.String(),
    Action:         string(constants.ActionAutoGenerated),
    ClientType:     string(constants.ClientTypeSystem),
})
```

### Log Only After a Successful Transaction

When the mutation runs inside a transaction (`txManager.RunInTx`), **call `Log` only after the transaction has committed successfully** — i.e. outside the transaction callback, after checking that `RunInTx` returned no error.

**Why:** If you log inside the callback and the transaction is rolled back later (e.g. a subsequent step in the same callback fails), you would have written an audit entry for a change that never persisted. Audit logs must reflect committed state only.

**Pattern (see `pkg/services/site/site_actions.go`):**

1. Run all DB work inside `RunInTx(ctx, func(txCtx context.Context) error { ... })`.
2. If `RunInTx` returns an error, return that error to the caller — do not log.
3. Only after a successful return from `RunInTx`, call `s.auditLogger.Log(ctx, Entry{...})` using data produced by the transaction (e.g. created entity ID).

```go
var createdSite *schemas.Site

err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    // All DB work inside the transaction. Use txCtx for repo calls.
    if err := s.siteRepository.CreateSite(txCtx, siteToCreate); err != nil {
        return err
    }
    createdSite = siteToCreate
    return s.organizationSiteRepository.BindSiteToOrganization(txCtx, orgSiteToCreate)
})
if err != nil {
    return nil, err
}

// Only after successful commit: audit log.
s.auditLogger.Log(ctx, auditlog.Entry{
    OrganizationID: orgID,
    Feature:        constants.FeatureSiteManagement.Value,
    EntityType:     string(constants.EntityTypeSite),
    EntityID:       createdSite.ID.String(),
    Action:         string(constants.ActionCreated),
    NewValue:       map[string]any{"name": req.Name, ...},
})

// Then cache invalidation, return, etc.
```

**Summary:**

| Do | Don't |
|----|--------|
| Call `Log` after `RunInTx` returns with no error | Call `Log` inside the `RunInTx` callback |
| Use committed data (e.g. `createdSite.ID`) in the entry | Log before the transaction commits |
| Return on tx error without logging | Log when any step (including tx) failed |
| For user actions, include `UserID` in the entry (pass userID from handler into service) | Omit UserID for HTTP/device actions — required for attribution |

## Metadata

The `Metadata` field (`map[string]any`) carries contextual information that doesn't fit into `OldValue`/`NewValue`. While `OldValue`/`NewValue` track **state changes** (before/after), `Metadata` captures **context about the action itself** — why it happened, where, or supplementary identifiers.

### When to Use Metadata

| Use Metadata for | Use OldValue/NewValue for |
|-----------------|--------------------------|
| Action context: email, locale, platform_code | State before/after a mutation |
| Supplementary identifiers: device_id, platform_id | Entity field values that changed |
| Classification: `"type": "otp_generated"` | The actual data that was created/updated/deleted |
| Geospatial data: latitude, longitude | Previous vs new field values |
| Operational flags: `"sessions_revoked": true` | N/A (flags that describe the action go in Metadata) |

### Metadata Patterns by Domain

**Authentication** — include identifying context (who tried to do what):

```go
// Sign-in (success or failure)
Metadata: map[string]any{"email": email, "platform_code": platformCode}

// Sign-out
Metadata: map[string]any{"email": claims.Email, "organization_name": claims.OrganizationName}

// Choose organization
Metadata: map[string]any{"email": claims.Email, "organization_name": orgUser.Organization.Name}

// Forgot password
Metadata: map[string]any{"type": "forgot_password_requested", "locale": locale}

// Reset password
Metadata: map[string]any{"type": "password_reset_completed", "email": user.Email, "sessions_revoked": true}
```

**Organization management** — include org-level attributes:

```go
// Create organization
Metadata: map[string]any{"locale": organization.Locale, "timezone": organization.Timezone, "is_admin": organization.IsAdmin}

// Delete organization
Metadata: map[string]any{"industry": string(org.Industry), "size": string(org.Size)}
```

**FTM (device operations)** — include device/user identifiers and operation type:

```go
// OTP generation
Metadata: map[string]any{"type": "otp_generated", "expires_at": newOtp.ExpiresAt}

// Device initialization
Metadata: map[string]any{"type": "device_initialized", "os": hostInfo.OS, "hostname": hostInfo.Hostname, "arch": hostInfo.Arch}

// Delete completed process
Metadata: map[string]any{"type": "completed_process_deleted", "device_id": deviceID.String()}
```

**Notification actions** — include the platform scope:

```go
Metadata: map[string]any{"platform_id": platformID.String()}
```

**Platform management** — include identifying attributes:

```go
Metadata: map[string]any{"platform_code": platform.Code, "platform_name": platform.Name}
```

**Site management** — include geospatial data:

```go
Metadata: map[string]any{"latitude": req.Latitude, "longitude": req.Longitude}
```

**Identity / Access** — include the affected user or role details:

```go
// Delete user
Metadata: map[string]any{"user_name": orgUser.User.Name, "user_email": orgUser.User.Email}

// Delete role
Metadata: map[string]any{"description": existingRole.Description}
```

### Metadata Rules

1. **Keep it flat** — use simple key-value pairs, not nested objects. The audit log UI renders metadata as a flat table.
2. **Use snake_case keys** — consistent with the rest of the codebase (`"platform_code"`, not `"platformCode"`).
3. **No sensitive data** — metadata is NOT auto-scrubbed like `OldValue`/`NewValue`. Never put passwords, tokens, or secrets in metadata.
4. **No redundant data** — don't repeat data already in other entry fields. If `EntityID` holds the device ID, don't also put `"device_id"` in metadata (unless it's a *different* device than the entity).
5. **Stringify UUIDs** — always call `.String()` on UUIDs before adding to metadata: `"platform_id": platformID.String()`.
6. **Use `"type"` for classification** — when the same action constant covers multiple operations (e.g., `ActionCreated` for both OTP generation and device initialization), add a `"type"` key to distinguish them.
7. **Include searchable context** — think about what an auditor would search for. Email addresses, platform codes, and operation types are useful; internal implementation details are not.

## Constants

Use constants from `pkg/database/constants/` for all Feature, EntityType, and Action values. Never use raw strings.

### Features

Defined in `pkg/database/constants/features.go`. Each feature has a `Value`, `FeatureColor` (for UI), and `TranslationKey`.

| Constant | Value | Use for |
|----------|-------|---------|
| `FeatureAccessControlManagement` | `"access_control_management"` | Authentication (login, logout, forgot/reset password, choose org), roles, permissions |
| `FeatureFTM` | `"ftm_module"` | FTM operations: devices, processes, quotas, OTP, device logs |
| `FeatureSiteManagement` | `"site_management"` | Site CRUD |
| `FeatureBilling` | `"billing"` | Invoices, payments, subscriptions, billing contacts |
| `FeaturePlatformManagement` | `"platform_management"` | Platform activation/deactivation |
| `FeatureNotification` | `"notification"` | Notification read/delete actions |
| `FeatureOrganizationManagement` | `"organization_management"` | Organization CRUD |
| `FeatureUnknown` | `"unknown"` | Fallback for legacy/unmapped values |

Usage: `constants.FeatureBilling.Value`

### Entity Types

Defined in `pkg/database/constants/entity_types.go`. Each has a typed constant (`EntityType`) and an info struct (`EntityTypeInfo`) with a translation key.

| Constant | Value |
|----------|-------|
| `EntityTypeProcess` | `"process"` |
| `EntityTypeDevice` | `"device"` |
| `EntityTypeUser` | `"user"` |
| `EntityTypeOrganization` | `"organization"` |
| `EntityTypeRole` | `"role"` |
| `EntityTypePermission` | `"permission"` |
| `EntityTypeSession` | `"session"` |
| `EntityTypeConfiguration` | `"configuration"` |
| `EntityTypeSubscriptionPlanTemplate` | `"subscription_plan_template"` |
| `EntityTypeInvoice` | `"invoice"` |
| `EntityTypePaymentProofReviewer` | `"payment_proof_reviewer"` |
| `EntityTypePaymentMethod` | `"payment_method"` |
| `EntityTypePaymentProof` | `"payment_proof"` |
| `EntityTypeBillingContact` | `"billing_contact"` |
| `EntityTypePayment` | `"payment"` |
| `EntityTypeRefund` | `"refund"` |
| `EntityTypePaymentMethodConfig` | `"payment_method_config"` |
| `EntityTypeAccessCard` | `"access_card"` |
| `EntityTypeBackgroundProcess` | `"background_process"` |
| `EntityTypePlatform` | `"platform"` |
| `EntityTypeSite` | `"site"` |
| `EntityTypeNotification` | `"notification"` |
| `EntityTypeQuota` | `"quota"` |

Usage: `string(constants.EntityTypeInvoice)`

### Actions

Defined in `pkg/database/constants/actions.go`. **Keep actions minimal and consistent: CRUD + auth.**

| Constant | Value | Use for |
|----------|-------|--------|
| `ActionCreated` | `"created"` | Create (entity, upload, etc.) |
| `ActionUpdated` | `"updated"` | Update, approve, reject, revoke/unrevoke, cancel, set default, send, state changes |
| `ActionDeleted` | `"deleted"` | Delete |
| `ActionLogin` | `"login"` | Login, choose organization |
| `ActionLogout` | `"logout"` | Logout |
| `ActionReset` | `"reset"` | Password reset, forgot password |
| `ActionLoginFailed` | `"login_failed"` | Failed login attempt (security) |
| `ActionUnknown` | `"unknown"` | Fallback for display of legacy/unmapped values |

Usage: `constants.ActionCreated.Value`. Do not add new action types unless necessary; prefer mapping to one of the above (e.g. "approved" → `ActionUpdated`). Legacy values in the DB are mapped in `actionsByValue` for display/translation.

### Client Types

Defined in `pkg/database/constants/client_types.go`:

| Constant | Value | Use when |
|----------|-------|----------|
| `ClientTypeWeb` | `"web"` | Browser-based user action (default if empty) |
| `ClientTypeMobile` | `"mobile"` | Mobile app action |
| `ClientTypeDevice` | `"device"` | Generic device |
| `ClientTypeEdgeDevice` | `"edge_device"` | FTM edge device |
| `ClientTypeSystem` | `"system"` | Scheduled job, auto-generated |
| `ClientTypeAPI` | `"api"` | External API integration |

Usage: `string(constants.ClientTypeEdgeDevice)`

## Sensitive Data Scrubbing

`OldValue` and `NewValue` are automatically scrubbed before being persisted. Any top-level key matching the blacklist is replaced with `"[REDACTED]"`.

**Blacklisted keys** (case-insensitive, defined in `pkg/database/services/auditlog/sanitize.go`):

| Category | Keys |
|----------|------|
| Credentials | `password`, `secret`, `credential`, `private_key` |
| Tokens | `token`, `api_key`, `apikey`, `gateway_token`, `refresh_token`, `access_token`, `authorization` |
| Hashes & OTPs | `hash`, `otp`, `otp_hash` |
| Payment gateway | `gateway_transaction_id`, `gateway_refund_id` |

**Example:** If you pass:
```go
NewValue: map[string]any{
    "id": "abc-123",
    "gateway_token": "tok_secret_xyz",
    "card_last_four": "4242",
}
```

The stored JSON will be:
```json
{"id": "abc-123", "gateway_token": "[REDACTED]", "card_last_four": "4242"}
```

**Scope:** Only top-level keys are scrubbed. Deeply nested sensitive data is not detected — don't nest secrets inside maps you pass as old/new values.

**Adding a new sensitive key:** Edit the `sensitiveKeys` map in `sanitize.go`.

## Middleware (IP and User Agent)

`AuditContextMiddleware` in `pkg/common/initialization/echo.go` runs on every HTTP request. It extracts the client IP and User-Agent from the request and stores them in context:

```go
func AuditContextMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            ctx := c.Request().Context()
            ctx = auditlog.WithIPAddress(ctx, c.RealIP())
            ctx = auditlog.WithUserAgent(ctx, c.Request().UserAgent())
            c.SetRequest(c.Request().WithContext(ctx))
            return next(c)
        }
    }
}
```

The `AuditLogger.Log()` method reads these from context automatically. Services don't need to do anything — IP and User-Agent are captured transparently.

**Behind Kong:** `c.RealIP()` reads `X-Forwarded-For` / `X-Real-IP` headers that Kong sets when proxying. The real client IP is captured, not Kong's internal IP.

## Testing

Tests use a shared no-op implementation defined in `pkg/services/billing/audit_test_helper_test.go`:

```go
type noOpAuditLogger struct{}

func (n *noOpAuditLogger) Log(_ context.Context, _ auditlog.Entry) {}
```

A generated mock is also available at `pkg/database/services/auditlog/mocks/AuditLogger.go` for tests that need to assert on audit log calls.

**When to use which:**
- **No-op** — most tests. Audit logging is a side effect you don't care about.
- **Mock** — tests that verify specific audit entries are created (e.g., testing that `DeletePaymentMethod` logs the correct entity type and action).

## What NOT to Do

| Don't | Do instead |
|-------|-----------|
| Log in handlers | Log in service methods — services have the business context |
| Set IP/UserAgent manually | Let the middleware handle it via context |
| Handle `Log()` errors | `Log()` returns nothing — it's fire-and-forget |
| Use raw strings for Feature/EntityType/Action | Use `constants.FeatureBilling.Value`, `string(constants.EntityTypeInvoice)`, `string(constants.ActionCreated)` |
| Pass entire structs as OldValue/NewValue | Pass `map[string]any` with only the relevant fields |
| Pass sensitive data and hope | The sanitizer is a safety net, not a substitute for careful data selection |
| Create `AuditLogger` manually in services | Inject via FX — `auditlog.Module` provides it |
| Log failed operations | Only log after successful mutations. Failures return errors before reaching the log call. |
| Omit UserID for user-initiated actions | Pass `&userID` from handler (e.g. `claims.UserID`) into the service and set `Entry.UserID` |
| Use request org in audit when acting on another org's entity | Use the entity's owning organization in `Entry.OrganizationID` (e.g. `targetOrgID` from existing resource) so the audit is attributed to the correct tenant |
| Log for read-only or in-memory operations (e.g. GeneratePDF, export, GET) | Only log **persistence mutations** (create, update, delete, state changes). Read-only APIs and in-memory generation with no DB write do not require an audit entry |
| Put sensitive data in Metadata | Metadata is NOT auto-scrubbed. Only `OldValue`/`NewValue` are scrubbed. Keep secrets out of Metadata entirely |
| Leave Metadata empty when context is available | Populate Metadata with action context (email, platform_code, type, etc.) to make audit entries searchable and informative |
| Use `FeatureUnknown` for known domains | Every domain should have its own Feature constant. `FeatureUnknown` is only for legacy/unmapped values |

---

## Package Structure

```
pkg/database/services/auditlog/
  audit_logger.go   — Entry struct, AuditLogger interface, Log implementation
  context.go        — Context keys for IP, UserAgent, ClientType (set by middleware)
  sanitize.go       — Sensitive key blacklist and sanitizeValue function
  MODULE.go         — FX module (injects action_audit_logs.Repository, provides AuditLogger)
  mocks/            — Generated mock for testing
```

## Related

- `pkg/database/constants/` — Feature, EntityType, Action, ClientType constants
- `pkg/database/repositories/action_audit_logs/` — Repository that persists to `common.action_audit_logs`
- `pkg/database/schemas/common.go` — `ActionAuditLog` GORM schema
- `pkg/common/initialization/echo.go` — `AuditContextMiddleware`
