# Error Handling Rules

## The Rule

**Always use `common.CustomError`.** Never use `fmt.Errorf()`, `errors.New()`, or plain `error` returns.

Every error in this codebase must carry an error code, HTTP status, and a message ID for translation. Plain Go errors bypass the entire error pipeline (codes, translation, HTTP mapping) and produce unstructured responses.

## Creating Errors

### In Services and Common Packages

Use `common.NewCustomError` with the full fluent chain:

```go
return common.NewCustomError("Organization owner is required").
    WithMessageID("error_organization_owner_required").
    WithErrorCode(errorcodes.BadRequestForm).
    WithHTTPCode(http.StatusBadRequest)
```

Every error MUST have:
- `WithErrorCode()` — a constant from `pkg/common/errorcodes/`
- `WithHTTPCode()` — a constant from Go's `net/http` package (e.g., `http.StatusBadRequest`, `http.StatusNotFound`). **Never hardcode numeric status codes** (e.g., `WithHTTPCode(400)` is forbidden — use `WithHTTPCode(http.StatusBadRequest)`).
- `WithMessageID()` — a translation key that exists in `locales/en/messages.yaml` and `locales/id/messages.yaml`

### Wrapping Existing Errors

When an external call returns a plain `error`, wrap it:

```go
return common.NewCustomErrorFromError(err).
    WithMessageID("error_failed_to_hash_password").
    WithErrorCode(errorcodes.PasswordHashingError).
    WithHTTPCode(http.StatusInternalServerError)
```

### With Template Data

For dynamic messages with interpolation:

```go
return common.NewCustomError("User not found").
    WithMessageID("error_user_not_found_with_id").
    WithMessageData(map[string]interface{}{"userId": userID.String()}).
    WithErrorCode(errorcodes.UserNotFound).
    WithHTTPCode(http.StatusNotFound)
```

## In Repositories

### Single Record Queries (First)

Use `db_utils.HandleFindError` — it maps `gorm.ErrRecordNotFound` to a not-found error and other failures to a database error:

```go
if err := db.Where("id = ?", id).First(&record).Error; err != nil {
    return nil, db_utils.HandleFindError(err, "device_host",
        "Device host not found",
        "error_device_host_not_found",
        "error_retrieve_device_host_failed")
}
```

### List/Count Queries

Use `db_utils.HandleQueryError`:

```go
if err := db_utils.ExecuteCountQuery(query, &count, "devices"); err != nil {
    return nil, err
}
```

### Create/Update/Delete — Classify Database Errors

Use `db_utils.ClassifyDBError` to handle PostgreSQL constraint violations:

```go
if err := db.Create(&record).Error; err != nil {
    dbErr := db_utils.ClassifyDBError(err)
    if dbErr == nil {
        return db_utils.NewDatabaseError("Failed to create record", "error_create_failed")
    }
    switch dbErr.Type {
    case db_utils.DBErrorDuplicate:
        return db_utils.NewDuplicateError("Record already exists", "error_duplicate")
    case db_utils.DBErrorForeignKey:
        return db_utils.NewForeignKeyError("Referenced record not found", "error_fk_not_found")
    case db_utils.DBErrorNotNull:
        return db_utils.NewInvalidRequestError("Required field missing", "error_required_field")
    default:
        return db_utils.NewDatabaseError("Failed to create record", "error_create_failed")
    }
}
```

### Pre-built Repository Error Constructors

Use these from `db_utils` — they set the correct error code and HTTP status automatically:

| Constructor | Error Code | HTTP Status |
|-------------|-----------|-------------|
| `NewNotFoundError(msg, msgID)` | `SpecifiedResourceDoesNotExists` (2006) | `http.StatusNotFound` |
| `NewDuplicateError(msg, msgID)` | `DuplicateRecord` (2005) | `http.StatusConflict` |
| `NewDatabaseError(msg, msgID)` | `OtherDatabaseError` (2003) | `http.StatusInternalServerError` |
| `NewInvalidRequestError(msg, msgID)` | `InvalidRequest` (1005) | `http.StatusBadRequest` |
| `NewForeignKeyError(msg, msgID)` | `SpecifiedResourceDoesNotExists` (2006) | `http.StatusBadRequest` |
| `NewRaiseExceptionError(msg, msgID)` | `InvalidRequest` (1005) | `http.StatusBadRequest` |
| `NewDeadlineExceededError()` | `DeadlineExceeded` (1004) | `http.StatusGatewayTimeout` |

## Error Code Ranges

Codes follow the format `CCNNN` (category + number). Use the existing constant from `pkg/common/errorcodes/`:

| Range | Category |
|-------|----------|
| 10xx | General/System |
| 20xx | Database |
| 30xx | Authentication/Authorization |
| 40xx | User/Organization |
| 50xx | Password/Security |
| 60xx | Email |
| 70xx | Data Format/Parsing |
| 80xx | Cache/Redis |
| 90xx | Device |
| 100xx | MQTT |
| 110xx | Fuel Tank Monitoring |
| 120xx | WiFi |
| 130xx | External Services |
| 140xx | Site Management |
| 150xx | Soft Delete |
| 160xx | Notification |
| 170xx | Audit Logs |
| 180xx | Feature Management |
| 190xx | Platform/Subscriptions |
| 200xx | Storage |

When adding a new error, use the next available number in the appropriate category. If no category fits, create a new range.

## Checking Error Categories

Use the category helpers from `pkg/common/error.go` — never check error codes manually:

```go
if common.IsNotFoundError(err) { ... }
if common.IsDuplicateError(err) { ... }
if common.IsAuthError(err) { ... }
if common.IsValidationError(err) { ... }
if common.IsDatabaseError(err) { ... }
```

## Translation

Every `WithMessageID` key must have entries in both locale files:
- `pkg/common/translations/locales/en/messages.yaml`
- `pkg/common/translations/locales/id/messages.yaml`

If you add a new error with a `WithMessageID`, add the translation entry in both files.
