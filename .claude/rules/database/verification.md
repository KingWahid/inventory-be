---
paths:
  - "infra/database/migrations/**"
  - "infra/database/cmd/seed/mock-data/**"
  - "pkg/database/**"
---

# Database Verification

## Connecting to PostgreSQL

Use the credentials from `.env` when inspecting the dev database:

```bash
# Interactive psql session
docker exec -it industrix-postgres psql -U industrix_user -d industrix_db

# One-off query
docker exec -it industrix-postgres psql -U industrix_user -d industrix_db -c "SELECT ..."

# From host (if psql is installed locally)
PGPASSWORD=industrix_password psql -h localhost -p 5432 -U industrix_user -d industrix_db
```

Credentials (from `.env`):
- **User:** `POSTGRES_USERNAME` (default: `industrix_user`)
- **Password:** `POSTGRES_PASSWORD` (default: `industrix_password`)
- **Database:** `POSTGRES_DB` (default: `industrix_db`)
- **Host:** `localhost:5432` (from host) or `industrix-postgres:5432` (from within Docker network)

## Schema Migration Verification

When new migration files are created (`infra/database/migrations/`):

1. **Apply all pending migrations:**
   ```bash
   make -C infra/database up
   ```

2. **Test rollback** — roll back only the new migrations:
   ```bash
   make -C infra/database down n=<number_of_new_migrations>
   ```

3. **Reapply** — confirm migrations are re-entrant:
   ```bash
   make -C infra/database up
   ```

All three steps must succeed. If rollback fails, the `.down.sql` file has a bug.

### Counting New Migrations

Count the new `.up.sql` files that were created. Each pair (`.up.sql` + `.down.sql`) counts as one migration.

### Dirty State Recovery

If a migration fails halfway (DB is dirty):
```bash
make -C infra/database force n=<last_good_version>
make -C infra/database up
```

## Mock Data Verification

When mock data YAML files are created or modified (`infra/database/cmd/seed/mock-data/`):

1. **Apply mock data:**
   ```bash
   make -C infra/database seed-mock-data
   ```

2. **Test rollback:**
   ```bash
   make -C infra/database rollback-mock-data
   ```

3. **Reapply:**
   ```bash
   make -C infra/database seed-mock-data
   ```

All three steps must succeed.

### When Both Change

If both migrations and mock data change in the same task, verify migrations first (mock data depends on schema).

## Repository Tests

**Trigger:** When repository code changes (`pkg/database/`) **OR** when migrations are created or modified (`infra/database/migrations/`).

Migrations change the schema that repositories depend on. A migration that alters a column type, adds a constraint, or drops a default can silently break repository queries. Always re-run repository tests after migration changes — even if no repository code was modified.

**Unit tests (always run):**
```bash
go test -tags '!integration' ./pkg/database/repositories/{affected_entity}/...
```

**Integration tests (requires `industrix-postgres` running):**
```bash
go test -tags integration ./pkg/database/repositories/{affected_entity}/...
```

**When migrations change but no specific repository was modified**, run integration tests for all repositories that touch the affected tables:
```bash
go test -tags integration ./pkg/database/repositories/...
```

## Mock Regeneration Check

If a repository interface changed, mocks must be regenerated:
```bash
make mocks-generate
```

Verify no uncommitted changes to `*/mocks/Repository.go` files after regeneration.

### Mockery Version Requirement

This project requires **mockery v2.53.5 or later**. Versions before v2.53.5 (notably v2.52.1) have a known bug with `go.uber.org/fx` that causes all mock generation to fail with:

```
internal error: package "go.uber.org/fx" without types was imported from ...
```

If you see this error, upgrade mockery:
```bash
go install github.com/vektra/mockery/v2@v2.53.5
```
