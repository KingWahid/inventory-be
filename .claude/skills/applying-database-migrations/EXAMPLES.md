# Database Migration Examples

## Example 1: Testing a New Schema Migration

```bash
cd infra/database

# Apply
make up
# Expected: 000245_unlink_notifications_from_billing.up.sql applied

# Verify in DB (connect and check changes)

# Rollback
make down n=1
# Expected: 000245_unlink_notifications_from_billing.down.sql applied

# Reapply
make up
```

## Example 2: Rollback Multiple Migrations

```bash
cd infra/database

# Rollback last 3 migrations
make down n=3
# Sequentially rolls back: 245, 244, 243
```

## Example 3: Recovering from Dirty State

Migration failed halfway — DB is dirty:

```bash
cd infra/database

make up
# Error: Dirty database version 245. Fix and force version.

# Fix the SQL, then force to previous clean version
make force n=244

# Reapply
make up
```

## Example 4: Full Dev Setup from Scratch

```bash
cd infra/database

# One command: migrations + mock data + crons
make dev-setup
```

For migration file conventions (naming, SQL patterns, down migrations), see `.claude/rules/infra/naming.md`.
