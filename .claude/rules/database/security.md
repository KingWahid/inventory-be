---
paths:
  - "pkg/database/**"
---

# Database Security Rules

## Parameterized Queries

Always use placeholder queries. Never concatenate strings into SQL:

```go
// Correct
query.Where("user_id = ? AND deleted_at IS NULL", userID)

// Wrong — SQL injection risk
query.Where("user_id = '" + userID.String() + "' AND deleted_at IS NULL")
```

## Organization Scoping

All data access is scoped to an organization by default. Every query on a tenant-owned table MUST filter by `organization_id`:

```go
db.Where("organization_id = ? AND id = ? AND deleted_at IS NULL", orgID, deviceID).First(&device)
```

Unscoped access is the explicit exception — only for admin operations with clearly named methods (`*Unscoped`).

## Soft Deletes

Always filter `deleted_at IS NULL` in queries unless explicitly working with deleted records. When cleaning up in tests, use `.Unscoped()` to reach soft-deleted rows.

## UUID Validation

Parse and validate UUID strings before using them in queries:

```go
userID, err := uuid.Parse(userIDStr)
if err != nil {
    return common.NewCustomError("Invalid user ID").
        WithErrorCode(errorcodes.FailedToParseUUID).
        WithHTTPCode(http.StatusBadRequest)
}
```

## Error Classification

Use `db_utils.ClassifyDBError` after mutations to handle PostgreSQL constraint violations specifically — don't return raw database errors to callers. Raw errors may leak schema details.

## Sensitive Fields

Never return password hashes, tokens, or secrets in query results. Use `json:"-"` on schema fields that should not be serialized:

```go
Password string `gorm:"column:password;size:255" json:"-"`
```
