---
paths:
  - "pkg/database/**/*_test.go"
  - "pkg/database/**/*_integration_test.go"
---

# Database Layer Testing

## Unit Tests (`*_test.go`)

Build tag: `//go:build !integration`

Use testify/suite with sqlmock:

```go
type RepositoryTestSuite struct {
    suite.Suite
    repo         Repository
    db           *gorm.DB
    mock         sqlmock.Sqlmock
    cacheManager *caches.CacheManager
}

func (s *RepositoryTestSuite) SetupTest() {
    var sqlDB *sql.DB
    sqlDB, s.mock, _ = sqlmock.New()
    s.db, _ = gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
        SkipDefaultTransaction: true,
    })
    s.cacheManager = &caches.CacheManager{}
    s.repo, _ = NewRepository(s.db, s.cacheManager, false)
}

func TestRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(RepositoryTestSuite))
}
```

Mock SQL expectations with `regexp.QuoteMeta` for INSERT/SELECT and verify with `s.mock.ExpectationsWereMet()`.

Test cases per method:
- `Success` — happy path
- `DatabaseError` — DB returns error
- `NotFound` — record doesn't exist
- `ContextDeadlineExceeded` — timeout

## Integration Tests (`*_integration_test.go`)

Build tag: `//go:build integration || integration_all`

Use real PostgreSQL via environment variables (`TEST_DB_HOST`, `TEST_DB_PORT`, etc.):

```go
type RepositoryIntegrationTestSuite struct {
    suite.Suite
    repo   Repository
    db     *gorm.DB
    testOrgID uuid.UUID
}
```

**SetupSuite**: Connect to test DB, create test fixtures (org, user, device).
**TearDownSuite**: Delete fixtures in reverse dependency order using `.Unscoped()`.
**SetupTest / TearDownTest**: Clean test-specific data for isolation.

Always skip if DB is unavailable:
```go
if test_utils.SkipIfTestDBUnavailable(s.T(), host, port, "SuiteName") {
    return
}
```

### GORM `default:true` boolean columns

When a schema field has a GORM tag of `default:true`, GORM omits the field from the INSERT when the Go value is the zero value (`false`), so the database default (`true`) wins. Setting the field to `false` in the struct literal **will not** write `false`.

To force `false` in integration tests, use a two-step Create-then-Update pattern:

```go
role := &schemas.Role{
    ID:   uuid.New(),
    Name: "Non-Assignable Role",
    Type: string(constants.RoleTypeBuiltIn),
    // IsAssignable intentionally omitted — GORM would skip the zero value
    // and the DB default (true) would apply.
}
s.Require().NoError(s.db.Create(role).Error)
// Explicit UPDATE forces is_assignable = false.
s.Require().NoError(s.db.Model(role).Update("is_assignable", false).Error)
```

Currently affected field: `schemas.Role.IsAssignable` (`gorm:"default:true"`).

## Mock Generation

Tool: mockery v2 (v2.53.5+). Mocks live in `{entity}/mocks/Repository.go`. Never edit generated mock files.

### When to Regenerate

- After adding a new method to a repository interface
- After removing a method from a repository interface
- After changing a method signature (parameters, return types)
- After creating a new repository

### Commands

**All repositories at once (recommended):**
```bash
make mocks-generate
```

**Single repository:**
```bash
mockery --dir=./pkg/database/repositories/{entity} --name=Repository --output=./pkg/database/repositories/{entity}/mocks --outpkg=mocks
```

**Clean all mocks and regenerate:**
```bash
make mocks-clean && make mocks-generate
```

### How It Works

The Makefile in `pkg/database/repositories/` auto-discovers all directories containing a `repo.go` file and runs mockery for each. Currently 40+ repositories have generated mocks.

## Running Tests

**Unit tests (all repos):**
```bash
go test -tags '!integration' ./pkg/database/repositories/...
```

**Unit tests (single repo):**
```bash
go test -tags '!integration' ./pkg/database/repositories/{entity}/...
```

**Integration tests (requires running DB — see `.env.test`):**
```bash
go test -tags integration ./pkg/database/repositories/{entity}/...
```

**All integration tests:**
```bash
go test -tags integration_all ./pkg/database/repositories/...
```
