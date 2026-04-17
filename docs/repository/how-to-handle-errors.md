# How to Handle Errors

This guide covers error handling patterns, database error classification, and WithMessageID for i18n support.

## Error Handling Principles

All repository errors must:
1. **Wrap DB errors** in `common.NewCustomError(...)`
2. **Include `WithMessageID`** for frontend i18n translation
3. **Set appropriate `ErrorCode`** from `pkg/common/errorcodes`
4. **Attach HTTP status codes** as metadata
5. **Log with structured context** using `zap.S()`

## Query Helper Functions

Use shared helpers for common error patterns:

```go
// Single record retrieval (First)
HandleFindError(err error, entityName, notFoundMsg, notFoundMsgID, retrieveFailedMsgID string) error

// List queries (Find) - "not found" is not an error
HandleQueryError(err error, operation, entityName, messageID string) error
```

### HandleFindError Example

```go
func (r *repo) GetEntityByID(ctx context.Context, entityID uuid.UUID) (*schemas.Entity, error) {
    var entity schemas.Entity

    err := r.db.WithContext(ctx).
        Where("id = ? AND deleted_at IS NULL", entityID).
        First(&entity).Error

    if err != nil {
        return nil, HandleFindError(err, entityTableName,
            "Entity not found",                  // User message
            "error_entity_not_found",            // MessageID for frontend
            "error_retrieve_entity_failed")      // MessageID for DB error
    }

    return &entity, nil
}
```

**HandleFindError handles:**
- `gorm.ErrRecordNotFound` → 404 with `notFoundMsgID`
- `context.DeadlineExceeded` → 504 with `error_request_deadline_exceeded`
- Other errors → 500 with `retrieveFailedMsgID`

### HandleQueryError Example

```go
func (r *repo) ListEntities(ctx context.Context, page, limit *int) ([]schemas.Entity, *common.PaginationInfo, error) {
    var total int64

    query := r.db.WithContext(ctx).Model(&schemas.Entity{})

    if err := ExecuteCountQuery(query, &total, entityTableName); err != nil {
        return nil, nil, err  // ExecuteCountQuery uses HandleQueryError internally
    }
    // ...
}
```

**HandleQueryError handles:**
- `context.DeadlineExceeded` → 504 Gateway Timeout
- Other errors → 500 Internal Server Error
- **Does NOT** treat "not found" as an error (list can be empty)

## Database Error Classification

Use `ClassifyDBError` to identify PostgreSQL constraint violations:

### DBErrorType Constants

```go
const (
    DBErrorUnknown         DBErrorType = iota
    DBErrorDuplicate                   // 23505 - unique_violation
    DBErrorForeignKey                  // 23503 - foreign_key_violation
    DBErrorNotNull                     // 23502 - not_null_violation
    DBErrorCheckConstraint             // 23514 - check_violation
    DBErrorDataTooLong                 // 22001 - string_data_right_truncation
    DBErrorInvalidInput                // 22P02 - invalid_text_representation
    DBErrorExclusion                   // 23P01 - exclusion_violation
    DBErrorSerializationFailure        // 40001 - serialization_failure
    DBErrorDeadlock                    // 40P01 - deadlock_detected
    DBErrorUndefinedTable              // 42P01 - undefined_table
)
```

### DBErrorInfo Struct

```go
type DBErrorInfo struct {
    Type       DBErrorType
    Constraint string // e.g., "users_email_key"
    Column     string // e.g., "email"
    Detail     string // PostgreSQL detail message
    Table      string // e.g., "common.users"
}
```

## Dedicated Error Handlers for Write Operations

Create context-specific error handlers for Create and Update operations:

### Create Error Handler

```go
// handleCreateEntityError handles Create errors with context-specific messages.
func (r *repo) handleCreateEntityError(err error, entity *schemas.Entity) error {
    // Check deadline first (don't log - self-explanatory)
    if errors.Is(err, context.DeadlineExceeded) {
        return NewDeadlineExceededError()
    }

    // Classify database error for constraint violations
    if dbErr := ClassifyDBError(err); dbErr != nil {
        switch dbErr.Type {
        case DBErrorDuplicate:
            // Log with useful context
            zap.S().With(zap.Error(err), zap.String("email", entity.Email)).
                Warn("Duplicate user email")
            return NewDuplicateError(
                "A user with this email already exists",
                "error_user_duplicate")

        case DBErrorForeignKey:
            // Check which FK failed
            if strings.Contains(dbErr.Constraint, "organization_id") {
                zap.S().With(zap.Error(err), zap.String("org_id", entity.OrganizationID.String())).
                    Warn("Organization not found for user")
                return NewForeignKeyError(
                    "The selected organization does not exist",
                    "error_user_organization_not_found")
            }
            // Generic FK error
            zap.S().With(zap.Error(err), zap.String("constraint", dbErr.Constraint)).
                Warn("Foreign key violation")
            return NewForeignKeyError(
                "One of the selected items no longer exists",
                "error_entity_reference_not_found")

        case DBErrorNotNull:
            zap.S().With(zap.Error(err), zap.String("column", dbErr.Column)).
                Warn("Required field missing")
            return NewInvalidRequestError(
                "A required field is missing",
                "error_entity_required_field_missing")

        case DBErrorCheckConstraint, DBErrorDataTooLong, DBErrorInvalidInput:
            zap.S().With(zap.Error(err), zap.String("constraint", dbErr.Constraint)).
                Warn("Invalid input")
            return NewInvalidRequestError(
                "Invalid input data provided",
                "error_entity_invalid_input")

        case DBErrorSerializationFailure, DBErrorDeadlock:
            zap.S().With(zap.Error(err)).Warn("Transaction conflict")
            return NewDatabaseError(
                "Please try again",
                "error_transaction_conflict")

        case DBErrorUndefinedTable:
            // Schema/configuration error - deployment issue
            zap.S().With(zap.Error(err), zap.String("table", dbErr.Table)).
                Error("Database table does not exist")
            return NewDatabaseError(
                "Database configuration error",
                "error_database_schema_error")
        }
    }

    // Fallback for unexpected errors
    return NewDatabaseError("Failed to create entity", "error_create_entity_failed")
}
```

**Create handler needs entity data for logging context (email, IDs, etc.).**

### Update Error Handler

```go
// handleUpdateEntityError handles Update errors.
func (r *repo) handleUpdateEntityError(err error) error {
    if errors.Is(err, context.DeadlineExceeded) {
        return NewDeadlineExceededError()
    }

    if dbErr := ClassifyDBError(err); dbErr != nil {
        switch dbErr.Type {
        case DBErrorDataTooLong:
            // Most likely for varchar field updates
            zap.S().With(zap.Error(err), zap.String("column", dbErr.Column)).
                Warn("Data too long when updating entity")
            return NewInvalidRequestError(
                "The provided value is too long",
                "error_entity_data_too_long")

        case DBErrorDuplicate:
            // Unlikely for updates but handle defensively
            zap.S().With(zap.Error(err), zap.String("constraint", dbErr.Constraint)).
                Warn("Duplicate value when updating entity")
            return NewDuplicateError(
                "An entity with this value already exists",
                "error_entity_duplicate")

        case DBErrorCheckConstraint, DBErrorInvalidInput:
            zap.S().With(zap.Error(err), zap.String("constraint", dbErr.Constraint)).
                Warn("Invalid input when updating entity")
            return NewInvalidRequestError(
                "Invalid input data provided",
                "error_entity_invalid_input")

        case DBErrorSerializationFailure, DBErrorDeadlock:
            zap.S().With(zap.Error(err)).Warn("Transaction conflict")
            return NewDatabaseError(
                "Please try again",
                "error_transaction_conflict")
        }
    }

    return NewDatabaseError("Failed to update entity", "error_update_entity_failed")
}
```

**Update handler is simpler:**
- No entity parameter needed
- Focuses on `DBErrorDataTooLong` (varchar field updates)
- No `DBErrorForeignKey` or `DBErrorNotNull` (updates don't change FKs/required fields typically)

## Error Helper Functions

Use these builders for consistent error formatting:

```go
// Standard error helpers (from pkg/database/db_utils/errors.go)
NewDeadlineExceededError() *common.CustomError
NewDatabaseError(message, messageID string) *common.CustomError
NewDuplicateError(message, messageID string) *common.CustomError
NewInvalidRequestError(message, messageID string) *common.CustomError
NewForeignKeyError(message, messageID string) *common.CustomError
NewNotFoundError(message, messageID string) *common.CustomError
```

### Example Usage

```go
// Deadline exceeded (no logging needed)
if errors.Is(err, context.DeadlineExceeded) {
    return NewDeadlineExceededError()
}

// Duplicate record
return NewDuplicateError(
    "An entity with this ID already exists",
    "error_entity_duplicate")

// Not found
return NewNotFoundError("Entity not found", "error_entity_not_found")

// Generic database error
return NewDatabaseError("Failed to retrieve entity", "error_retrieve_entity_failed")
```

## MessageID Conventions

| Error Type | MessageID Pattern | Example |
|------------|-------------------|---------|
| DB connection nil | `error_db_connection_nil` | Initialization |
| Resource not found | `error_{resource}_not_found` | `error_user_not_found` |
| Duplicate record | `error_{resource}_duplicate` | `error_email_duplicate` |
| Validation failed | `error_{field}_invalid` | `error_email_invalid` |
| Request timeout | `error_request_deadline_exceeded` | Deadline exceeded |
| Database operation | `error_{operation}_{resource}_failed` | `error_create_user_failed` |
| Foreign key violation | `error_{resource}_{fk}_not_found` | `error_user_organization_not_found` |
| Required field missing | `error_{resource}_required_field_missing` | `error_user_required_field_missing` |
| Invalid input | `error_{resource}_invalid_input` | `error_user_invalid_input` |
| Transaction conflict | `error_transaction_conflict` | Serialization/deadlock |

## Logging Best Practices

### DO Log

✅ Constraint violations with context:
```go
zap.S().With(zap.Error(err), zap.String("email", user.Email)).
    Warn("Duplicate user email")
```

✅ Unexpected database errors:
```go
zap.S().With(zap.Error(err), zap.String("user_id", userID.String())).
    Error("Database error retrieving user")
```

✅ Context that helps debugging:
```go
zap.S().With(zap.Error(err), zap.String("constraint", dbErr.Constraint)).
    Warn("Foreign key violation")
```

### DON'T Log

❌ Deadline exceeded (self-explanatory, adds noise):
```go
// DON'T DO THIS
if errors.Is(err, context.DeadlineExceeded) {
    zap.S().Error("Request deadline exceeded")  // Remove this
    return NewDeadlineExceededError()
}

// CORRECT
if errors.Is(err, context.DeadlineExceeded) {
    return NewDeadlineExceededError()  // No logging
}
```

❌ Errors without additional context:
```go
// DON'T DO THIS
zap.S().With(zap.Error(err)).Error("Database error")  // No useful context
```

❌ Duplicate information already in error message:
```go
// DON'T DO THIS
zap.S().Error("User not found")  // Redundant with error message
return NewNotFoundError("User not found", "error_user_not_found")
```

## Error Code Quick Reference

| DBErrorType | Typical MessageID | HTTP Code |
|-------------|-------------------|-----------|
| `DBErrorDuplicate` | `error_{resource}_duplicate` | 409 Conflict |
| `DBErrorForeignKey` | `error_{resource}_{fk}_not_found` | 400 Bad Request |
| `DBErrorNotNull` | `error_{resource}_required_field_missing` | 400 Bad Request |
| `DBErrorCheckConstraint` | `error_{resource}_invalid_input` | 400 Bad Request |
| `DBErrorDataTooLong` | `error_{resource}_invalid_input` | 400 Bad Request |
| `DBErrorInvalidInput` | `error_{resource}_invalid_input` | 400 Bad Request |
| `DBErrorSerializationFailure` | `error_transaction_conflict` | 500 Internal |
| `DBErrorDeadlock` | `error_transaction_conflict` | 500 Internal |
| `DBErrorUndefinedTable` | `error_database_schema_error` | 500 Internal |

## Complete Error Handling Example

```go
import "github.com/industrix-id/backend/pkg/database/transaction"

func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)

    if err := db.Create(user).Error; err != nil {
        return r.handleCreateUserError(err, user)
    }
    return nil
}

func (r *userRepository) handleCreateUserError(err error, user *schemas.User) error {
    if errors.Is(err, context.DeadlineExceeded) {
        return NewDeadlineExceededError()
    }

    if dbErr := ClassifyDBError(err); dbErr != nil {
        switch dbErr.Type {
        case DBErrorDuplicate:
            zap.S().With(zap.Error(err), zap.String("email", user.Email)).
                Warn("Duplicate user email")
            return NewDuplicateError(
                "A user with this email already exists",
                "error_user_duplicate")

        case DBErrorForeignKey:
            zap.S().With(zap.Error(err), zap.String("org_id", user.OrganizationID.String())).
                Warn("Organization not found for user")
            return NewForeignKeyError(
                "The selected organization does not exist",
                "error_user_organization_not_found")

        case DBErrorNotNull:
            zap.S().With(zap.Error(err), zap.String("column", dbErr.Column)).
                Warn("Required field missing for user")
            return NewInvalidRequestError(
                "A required field is missing",
                "error_user_required_field_missing")

        case DBErrorSerializationFailure, DBErrorDeadlock:
            zap.S().With(zap.Error(err)).Warn("Transaction conflict")
            return NewDatabaseError(
                "Please try again",
                "error_transaction_conflict")
        }
    }

    return NewDatabaseError("Failed to create user", "error_create_user_failed")
}
```

## Security: Never Reveal Internal Details

❌ **NEVER** expose internal database details in error messages:

```go
// DON'T DO THIS
return fmt.Errorf("Table 'common.users' does not exist in schema 'public'")
```

✅ **DO** use generic yet helpful messages:

```go
// CORRECT
return NewDatabaseError(
    "Database configuration error",
    "error_database_schema_error")
```

**Why?**
- Prevents information disclosure to potential attackers
- Avoids confusing end users with technical details
- Logs contain full context for debugging

## Next Steps

- [How to Handle Transactions](./how-to-handle-transactions.md) - Write operations
- [How to Invalidate Caches](./how-to-invalidate-caches.md) - Cache strategies
