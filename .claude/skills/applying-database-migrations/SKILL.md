---
name: applying-database-migrations
description: Apply database schema migrations to test new migration files. Use proactively after creating schema migrations in infra/database/migrations/.
---

# Apply Database Migrations

For migration file conventions (naming, SQL patterns, seed data), see `.claude/rules/infra/naming.md`.

## Commands

All commands run from `infra/database/`:

```bash
cd infra/database

# Apply all pending migrations
make up

# Rollback last n migrations
make down n=<number>

# Force migration version (use when migration state is dirty)
make force n=<version_number>

# Full dev setup: migrations + mock data + crons
make dev-setup
```

The `migrate` tool uses `golang-migrate`. Install with:
```bash
make install
```

## Testing New Migrations

Always test both directions:

```bash
cd infra/database

# 1. Apply migration
make up

# 2. Verify in database (connect and check schema changes)

# 3. Test rollback
make down n=1

# 4. Verify rollback restored previous state

# 5. Reapply to keep DB in sync
make up
```

## Troubleshooting

**Dirty migration state** — migration failed halfway, DB is now in a dirty state:
```bash
# Option 1: Force to the last known-good version
make force n=<last_good_version>

# Option 2: Clean dirty state via SQL
make clean-dirty
```

**Migration already applied** — rollback first, then reapply:
```bash
make down n=1
make up
```

**Connection error** — check DB is running and env vars are correct:
```bash
make show-config        # shows current DB_HOST, DB_PORT, DB_NAME, DB_USER
make test-connection    # tests connectivity
```

## Important Notes

- Never modify migrations that have been deployed to staging/production
- Both `.up.sql` and `.down.sql` are required for every migration
- Database must be running locally (`docker ps` to verify)
- The `search_path` includes `public` and `extensions` (for PostGIS types)
- Seed data (permissions, roles, config) is part of SQL migrations — no separate seed step needed
