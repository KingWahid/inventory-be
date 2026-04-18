---
paths:
  - "pkg/database/**"
---

# Database Query Rules

## Qualify Column Names in JOINs

When a GORM query uses `.Joins()`, **always qualify column names with the full table name** in `.Where()`, `.Order()`, and `.Select()` clauses. Unqualified columns cause PostgreSQL `SQLSTATE 42702` ("column reference is ambiguous") when joined tables share column names.

Common columns shared across tables: `is_active`, `deleted_at`, `created_at`, `updated_at`, `id`, `name`.

```go
// Correct — fully qualified
db.Where("common.organization_users.user_id = ? AND common.organization_users.is_active = ?", userID, true).
    Joins("JOIN common.organizations o ON o.id = common.organization_users.organization_id AND o.is_active = ?", true).
    Find(&orgUsers)

// Wrong — ambiguous is_active (both tables have it)
db.Where("user_id = ? AND is_active = ?", userID, true).
    Joins("JOIN common.organizations o ON o.id = common.organization_users.organization_id AND o.is_active = ?", true).
    Find(&orgUsers)
```

This applies even if the query works today — a future migration adding a same-named column to the joined table will silently break it.

**Exception:** Queries without `.Joins()` don't need qualification since GORM scopes to the model's table automatically.

## Conditional Joins Require Pre-Emptive Qualification

When a query builder **conditionally** adds joins based on sort/filter flags, every `.Where()`, `.Order()`, and `search`/`LIKE` predicate in the *shared* path must be qualified up front — not only when the join is active. Otherwise a request with a specific `sort` or filter value can turn a passing query into a PostgreSQL `SQLSTATE 42702` at runtime.

```go
// buildFilterQuery is shared between the count and data queries.
// Sort switches downstream may add LEFT JOINs to `common.platforms` or
// `billing.subscription_plan_template_translations`, both of which have
// `deleted_at`, `code`, and `name` columns.
query := dbConn.WithContext(ctx).
    Model(&schemas.SubscriptionPlanTemplate{}).
    Where("billing.subscription_plan_templates.deleted_at IS NULL") // qualified even though no join here yet

if params.search != "" {
    like := "%" + strings.ToLower(params.search) + "%"
    query = query.Where(`
LOWER(billing.subscription_plan_templates.code) LIKE ? OR EXISTS (
    SELECT 1
    FROM billing.subscription_plan_template_translations t
    WHERE t.subscription_plan_template_id = billing.subscription_plan_templates.id
      AND t.deleted_at IS NULL
      AND LOWER(t.name) LIKE ?
)`, like, like)
}
```

Real case: `pkg/database/repositories/subscription_plan_templates/repo.go` — `sort=platform` conditionally adds `LEFT JOIN common.platforms p`. Both the templates table and `common.platforms` have a `code` column, so the shared `LOWER(code) LIKE ?` in the search clause had to be qualified as `LOWER(billing.subscription_plan_templates.code) LIKE ?` even though the JOIN only appears for one specific sort value. The bug does not manifest under any other sort value, which makes it a landmine that waits for a specific query-string combination to trigger.
