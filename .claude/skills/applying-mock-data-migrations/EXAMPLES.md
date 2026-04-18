# Mock Data Migration Examples

For YAML structure conventions (operations, UUID ranges, directory layout), see `.claude/rules/infra/naming.md`.

## Example 1: Apply and Verify New Mock Data

```bash
cd infra/database

# Apply
make seed-mock-data
# Expected: ✓ Applying migration: 0024_notifications_translations.yaml
#           ✓ Migration applied successfully

# Verify in database
# Query: SELECT * FROM common.notifications WHERE type = 'test_notification'

# Test rollback
make rollback-mock-data
# Expected: ✓ Rolling back migration: 0024_notifications_translations.yaml
#           ✓ Rollback successful

# Reapply
make seed-mock-data
```

## Example 2: Full Dev Setup from Scratch

```bash
cd infra/database

# One command does everything
make dev-setup
# Runs: make up → make seed-mock-data → make cron-up
```

## Example 3: Reset All Mock Data

When mock data is in a bad state:

```bash
cd infra/database

# Nuclear option: rollback everything, then re-seed
make reset-mock
# Step 1: Rolling back cron migrations...
# Step 2: Rolling back all data migrations...
# Step 3: Re-seeding data (force mode)...
# Step 4: Re-applying cron migrations...
# Mock data reset complete!
```

## Example 4: Apply with Crons vs Data Only

```bash
cd infra/database

# Data + crons (typical for full dev environment)
make seed-mock

# Data only (when you only need test records, no scheduled jobs)
make seed-mock-data

# Crons only (when you need scheduled jobs but already have data)
make cron-up
```

## Example 5: Handling Duplicate Key Error

```bash
cd infra/database

make seed-mock-data
# Error: duplicate key value violates unique constraint

# Rollback the failing migration
make rollback-mock-data

# Fix the YAML file (change conflicting IDs)

# Reapply
make seed-mock-data
```

## Example 6: Force Apply (CI/Automation)

```bash
cd infra/database

# Skips dirty state checks, upserts on conflict
make seed-mock-force
```
