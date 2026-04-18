# Unit Test Conventions

Comprehensive unit testing guidelines for the Industrix backend using `github.com/stretchr/testify/suite`.

## Table of Contents

- [Overview](#overview)
- [Test Organization](#test-organization)
- [Test Suite Structure](#test-suite-structure)
- [Test Naming Conventions](#test-naming-conventions)
- [Mock Management](#mock-management)
- [Integration Test Setup Patterns](#integration-test-setup-patterns)
- [Database Testing Modes](#database-testing-modes)
- [Test Patterns](#test-patterns)
- [Assertions](#assertions)
- [Test Helpers](#test-helpers)
- [Documentation Standards](#documentation-standards)
- [Coverage Requirements](#coverage-requirements)
- [Build Tags](#build-tags)
- [Running Tests](#running-tests)
- [Best Practices](#best-practices)

## Overview

Unit tests are critical for maintaining code quality, preventing regressions, and documenting expected behavior. This document establishes conventions for writing consistent, maintainable tests across the codebase.

### Testing Framework

We use **testify/suite** for structured test organization:
- `github.com/stretchr/testify/suite` - Test suite framework
- `github.com/stretchr/testify/assert` - Assertions
- `github.com/stretchr/testify/require` - Assertions that stop test execution on failure
- `github.com/stretchr/testify/mock` - Mocking framework
- `github.com/DATA-DOG/go-sqlmock` - SQL mock driver for database tests

### Coverage Requirements

| Layer | Minimum Coverage | Notes |
|-------|-----------------|-------|
| Repository | 80% | Critical data access layer |
| Service | 80% | Business logic layer |
| API/Controller | Not required | Focus on integration tests instead |
| Common/Utils | Critical functions only | Security, validation, encryption functions |

Check coverage: `make test-coverage`

## Test Organization

### File Location

Test files live **alongside source files** in the same directory:

```
pkg/database/repositories/
├── user.go           # Source code
├── user_test.go      # Tests for user.go
├── role.go
├── role_test.go
└── mocks/            # Generated mocks
    ├── UserRepository.go
    └── RoleRepository.go
```

### Service Layer Test Organization (Special Pattern)

**For service layer tests ONLY**, separate test suite definition from actual test cases:

```
pkg/services/access/
├── MODULE.go            # Uber FX module definition
├── SERVICE.go           # Service implementation
├── service_test.go      # Test suite definition, setup/teardown, helper functions
├── list_roles.go        # Role listing methods
├── roles_test.go        # Actual test cases for role methods
├── permissions.go       # Permission-related service methods
├── permissions_test.go  # Actual test cases for permissions.go
└── helpers.go
```

**Why this pattern for services?**
- Service layer has many methods across multiple files
- Avoids massive single test file (1000+ lines)
- Shared test suite and helpers reduce duplication
- Each `*_test.go` file maps to its source file
- Better organization for complex service logic

**`service_test.go` contains:**
```go
import (
    txMocks "github.com/industrix-id/backend/pkg/database/transaction/mocks"
    repoMocks "github.com/industrix-id/backend/pkg/database/repositories/organization_users/mocks"
    jwtMocks "github.com/industrix-id/backend/pkg/common/jwt/mocks"
)

// Test suite definition
type ServiceTestSuite struct {
    suite.Suite
    service         *service
    mockTxManager   *txMocks.Manager  // Centralized transaction mock
    mockOrgUserRepo *repoMocks.Repository
    mockJWTEncoder  *jwtMocks.Encoder
    // ... other mocks
}

// Setup and teardown
func (s *ServiceTestSuite) SetupSuite() { /* ... */ }
func (s *ServiceTestSuite) SetupTest() {
    s.mockTxManager = txMocks.NewManager(s.T())
    s.mockOrgUserRepo = repoMocks.NewRepository(s.T())
    s.mockJWTEncoder = jwtMocks.NewEncoder(s.T())

    // Default: allow transactions and execute the function
    s.mockTxManager.On("RunInTx", mock.Anything, mock.Anything).Return(nil).Maybe()

    s.service = &service{
        txManager:            s.mockTxManager,
        organizationUserRepo: s.mockOrgUserRepo,
        jwtEncoder:           s.mockJWTEncoder,
    }
}
func (s *ServiceTestSuite) TearDownTest() { /* ... */ }

// Test runner
func TestServiceTestSuite(t *testing.T) {
    suite.Run(t, new(ServiceTestSuite))
}

// Helper functions shared across all test files
func (s *ServiceTestSuite) createTestPaginationInfo(page, limit, total int) *common.PaginationInfo
func (s *ServiceTestSuite) createTestOrganizationUser(...) schemas.OrganizationUser
func (s *ServiceTestSuite) createTestJWTClaims(...) *jwt.AuthenticationTokenClaims
```

**Important**: The transaction manager mock is centralized at `pkg/database/transaction/mocks/Manager.go`. Do NOT define a local `mockTransactionManager` in `service_test.go`.

**`users_test.go` contains:**
```go
// Actual test methods for Users() and UsersByAuthToken()
func (s *ServiceTestSuite) TestUsers() {
    s.Run("Success", func() { /* test logic */ })
    s.Run("EmptyResult", func() { /* test logic */ })
    // ... more subtests
}

func (s *ServiceTestSuite) TestUsersByAuthToken() {
    s.Run("Success", func() { /* test logic */ })
    // ... more subtests
}
```

**⚠️ Repository tests use standard single-file pattern:**
```
pkg/database/repositories/
├── user.go
├── user_test.go         # Suite definition + all tests in ONE file
```

### Suite Per Type

Each struct/interface gets its own test suite:

```go
// One suite per repository
type UserRepositoryTestSuite struct {
    suite.Suite
    repo         repository.UserRepository
    db           *gorm.DB
    mock         sqlmock.Sqlmock
    cacheManager *caches.CacheManager
}

// One suite per service
type UserServiceTestSuite struct {
    suite.Suite
    service      service.UserService
    mockUserRepo *mocks.UserRepository
    mockRoleRepo *mocks.RoleRepository
    mockJWT      *mocks.JWTEncoder
}
```

### File Naming

Test files use `<source>_test.go` pattern:
- `user.go` -> `user_test.go`
- `auth_service.go` -> `auth_service_test.go`
- `validation.go` -> `validation_test.go`

## Test Suite Structure

### Basic Suite Template

```go
package repository

import (
    "context"
    "database/sql"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/stretchr/testify/suite"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

// UserRepositoryTestSuite defines test suite for UserRepository
type UserRepositoryTestSuite struct {
    suite.Suite
    repo UserRepository
    db   *gorm.DB
    mock sqlmock.Sqlmock
}

// SetupSuite runs once before all tests
func (s *UserRepositoryTestSuite) SetupSuite() {
    // Initialize database mock
    var err error
    var sqlDB *sql.DB
    sqlDB, s.mock, err = sqlmock.New()
    s.Require().NoError(err)

    s.db, err = gorm.Open(postgres.New(postgres.Config{
        Conn: sqlDB,
    }), &gorm.Config{})
    s.Require().NoError(err)

    // Initialize repository
    s.repo, err = NewUserRepository(s.db, nil, false)
    s.Require().NoError(err)
}

// TearDownSuite runs once after all tests
func (s *UserRepositoryTestSuite) TearDownSuite() {
    // Cleanup resources if needed
}

// TestUserRepositoryTestSuite runs the test suite
func TestUserRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(UserRepositoryTestSuite))
}
```

### Suite Lifecycle Methods

| Method | When | Use Case |
|--------|------|----------|
| `SetupSuite()` | Once before all tests | Initialize expensive resources (DB, mocks) |
| `TearDownSuite()` | Once after all tests | Clean up resources |
| `SetupTest()` | Before each test | Reset state between tests (if needed) |
| `TearDownTest()` | After each test | Clean up test-specific state |

**Recommendation**: Use `SetupSuite/TearDownSuite` for shared resources to improve performance.

## Test Naming Conventions

### Method Naming

Use `TestMethodName` with subtests for different scenarios:

```go
func (s *UserRepositoryTestSuite) TestGetUserByID() {
    userID := uuid.New()

    s.Run("Success", func() {
        // Arrange
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).
                AddRow(userID, "user@example.com"))

        // Act
        user, err := s.repo.GetUserByID(context.Background(), userID)

        // Assert
        s.NoError(err)
        s.Equal(userID, user.ID)
        s.Equal("user@example.com", user.Email)
    })

    s.Run("NotFound", func() {
        // Arrange
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnError(gorm.ErrRecordNotFound)

        // Act
        user, err := s.repo.GetUserByID(context.Background(), userID)

        // Assert
        s.Error(err)
        s.Nil(user)
        s.Contains(err.Error(), "not found")
    })

    s.Run("DatabaseError", func() {
        // Arrange
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnError(errors.New("connection lost"))

        // Act
        user, err := s.repo.GetUserByID(context.Background(), userID)

        // Assert
        s.Error(err)
        s.Nil(user)
    })
}
```

### Subtest Naming

Use descriptive names that explain the scenario:

**Good Examples:**
```go
s.Run("Success")
s.Run("NotFound")
s.Run("EmptyEmail")
s.Run("ContextTimeout")
s.Run("DuplicateKey")
s.Run("NilParameter")
```

**Bad Examples:**
```go
s.Run("Test1")        // Not descriptive
s.Run("Case2")        // Unclear what case this is
s.Run("Error")        // Too vague - what kind of error?
```

## Mock Management

### Mock Generation with mockery

Generate mocks using the Makefile target:
```bash
# Generate all repository mocks (recommended)
make mocks-generate
```

For manual generation (rarely needed):
```bash
# Install mockery
go install github.com/vektra/mockery/v2@latest

# Generate mock for a specific interface
mockery --name=Repository --dir=pkg/database/repositories/users --output=pkg/database/repositories/users/mocks
```

### Centralized Transaction Manager Mock

The `transaction.Manager` mock is **centralized** at `pkg/database/transaction/mocks/Manager.go`. This custom mock:

1. **Records expectations** using `testify/mock.Mock`
2. **Executes the transaction function** `fn(ctx)` after mock expectations are checked
3. **Auto-cleans up** via `NewManager(t)` which registers cleanup and assertion

**Why a custom mock?** Standard mockery-generated mocks don't execute callback functions. The transaction manager needs to both record expectations AND execute the business logic inside the transaction.

**Usage:**
```go
import txMocks "github.com/industrix-id/backend/pkg/database/transaction/mocks"

func (s *ServiceTestSuite) SetupTest() {
    s.mockTxManager = txMocks.NewManager(s.T())

    // Default: allow transactions and execute the function
    s.mockTxManager.On("RunInTx", mock.Anything, mock.Anything).Return(nil).Maybe()

    s.service = &service{
        txManager: s.mockTxManager,
        // ...
    }
}
```

**Do NOT:**
- Define local `mockTransactionManager` in `service_test.go`
- Create alternative transaction mocks in service packages

### sqlmock Expectations and GORM Query Patterns

**CRITICAL**: sqlmock expectations must EXACTLY match GORM's generated SQL queries. This section documents common patterns discovered through extensive testing.

#### Understanding GORM Query Generation

GORM transforms Go code into SQL queries with specific patterns that sqlmock must match precisely.

#### Key Pattern 1: LIMIT for First() Queries

**GORM's `First()` method ALWAYS adds `LIMIT 1` as the final argument:**

```go
✅ CORRECT:
s.mock.ExpectQuery(`SELECT \* FROM "common"\."users" WHERE \(id = \$1 AND deleted_at IS NULL\)`).
    WithArgs(userID, 1).  // LIMIT 1 is the second argument
    WillReturnRows(rows)

❌ WRONG:
s.mock.ExpectQuery(`SELECT \* FROM "common"\."users"`).
    WithArgs(userID).  // Missing LIMIT argument
    WillReturnRows(rows)
```

**Pattern applies to:**
- `db.First(&model, id)` → adds `LIMIT $N`
- `db.Where(...).First(&model)` → adds `LIMIT $N`
- Any query ending with `.First()`

#### Key Pattern 2: WHERE Clause Parentheses

**GORM wraps custom WHERE conditions in parentheses:**

```go
✅ CORRECT:
s.mock.ExpectQuery(`SELECT \* FROM "common"\."organizations" WHERE \(id = \$1 AND deleted_at IS NULL\)`).
    WithArgs(orgID, 1).
    WillReturnRows(rows)

❌ WRONG:
s.mock.ExpectQuery(`SELECT \* FROM "common"\."organizations" WHERE id = \$1 AND deleted_at IS NULL`).
    WithArgs(orgID, 1).
    WillReturnRows(rows)
```

**Pattern applies to:**
- Custom `Where()` clauses: `WHERE (custom_condition)`
- Soft delete checks: `WHERE (condition AND deleted_at IS NULL)`
- Combined conditions: `WHERE (cond1 AND cond2 AND ...)`

#### Key Pattern 3: Preload Queries

**GORM's `Preload()` triggers separate SELECT queries for relationships:**

```go
// Repository code
db.Preload("Site").First(&device, deviceID)

✅ CORRECT: Mock both queries
// Main query
s.mock.ExpectQuery(`SELECT \* FROM "common"\."devices"`).
    WithArgs(deviceID, 1).
    WillReturnRows(deviceRows)

// Preload query (triggered automatically)
s.mock.ExpectQuery(`SELECT \* FROM "operation"\."sites" WHERE "sites"\."id" = \$1`).
    WithArgs(siteID).
    WillReturnRows(siteRows)

❌ WRONG: Only mocking main query
s.mock.ExpectQuery(`SELECT \* FROM "common"\."devices"`).
    WithArgs(deviceID, 1).
    WillReturnRows(deviceRows)
// Missing: Preload query expectation
```

**Preload patterns:**
- Belongs-to: `SELECT * FROM "related_table" WHERE "related_table"."id" = $1`
- Has-many: `SELECT * FROM "related_table" WHERE "related_table"."foreign_key" IN ($1,$2,...)`
- Many-to-many: `SELECT * FROM "join_table" WHERE ...`

**Important**: Check schema names! Sites table is in `"operation"` schema, not `"common"`.

#### Key Pattern 4: IN Clause Expansion

**GORM expands IN clauses to individual arguments:**

```go
// Repository code
roleIDs := []uuid.UUID{role1, role2, role3}
db.Where("id IN ?", roleIDs)

✅ CORRECT: Each ID is a separate argument
s.mock.ExpectQuery(`SELECT \* FROM "common"\."roles" WHERE id IN \(\$1,\$2,\$3\)`).
    WithArgs(role1, role2, role3).  // Three separate arguments
    WillReturnRows(rows)

❌ WRONG: Slice as single argument
s.mock.ExpectQuery(`SELECT \* FROM "common"\."roles" WHERE id IN \(\$1\)`).
    WithArgs(roleIDs).  // Slice instead of individual args
    WillReturnRows(rows)
```

#### Key Pattern 5: Pagination Arguments

**GORM adds OFFSET and LIMIT as the final arguments:**

```go
// Repository code
db.Offset(10).Limit(20).Find(&results)

✅ CORRECT:
s.mock.ExpectQuery(`SELECT \* FROM "common"\."users"`).
    WithArgs(orgID, 10, 20).  // orgID, then offset, then limit
    WillReturnRows(rows)

❌ WRONG:
s.mock.ExpectQuery(`SELECT \* FROM "common"\."users"`).
    WithArgs(orgID).  // Missing offset and limit
    WillReturnRows(rows)
```

**Special case**: When offset is 0, GORM omits it:
```go
// Repository code with offset=0
db.Offset(0).Limit(10).Find(&results)

✅ CORRECT: GORM omits OFFSET 0
s.mock.ExpectQuery(`SELECT \* FROM "common"\."users"`).
    WithArgs(orgID, 10).  // Only limit, no offset
    WillReturnRows(rows)
```

#### Key Pattern 6: INSERT Column Order

**GORM sends INSERT columns in specific order: timestamps first, then other columns (unless timestamps are omitted):**

```go
// GORM generates:
// INSERT INTO "users" ("created_at","deleted_by","deleted_at","id","email","name",...) VALUES ($1,$2,$3,$4,$5,$6,...)

✅ CORRECT:
s.mock.ExpectQuery(`INSERT INTO "common"\."users"`).
    WithArgs(
        sqlmock.AnyArg(),  // created_at
        sqlmock.AnyArg(),  // deleted_by
        sqlmock.AnyArg(),  // deleted_at
        userID,            // id
        email,             // email
        name,              // name
        // ... other fields
    ).
    WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))

❌ WRONG: Wrong column order
s.mock.ExpectQuery(`INSERT INTO "common"\."users"`).
    WithArgs(userID, email, name, sqlmock.AnyArg()).  // Wrong order
    WillReturnRows(rows)
```

**Pattern**: `created_at, deleted_by, deleted_at` come first, then model fields in alphabetical order.

> **Note:** Some repositories explicitly manage timestamps (for example by omitting `created_at` / `updated_at`, or by defining models without those fields). In those cases GORM will _not_ include the timestamp columns in the generated INSERT. Always rely on the actual SQL shown in failed tests/logs when setting up sqlmock expectations, rather than assuming timestamps are present.

#### Key Pattern 7: UUID String Conversion

**GORM converts `uuid.UUID` to strings in WHERE clauses:**

```go
orgID := uuid.New()  // Type: uuid.UUID

✅ CORRECT: Use uuid.String() or direct UUID in mock
s.mock.ExpectQuery(`SELECT \* FROM "common"\."organizations"`).
    WithArgs(orgID).  // sqlmock handles UUID comparison
    WillReturnRows(rows)

// Or explicitly convert:
s.mock.ExpectQuery(`SELECT \* FROM "common"\."organizations"`).
    WithArgs(orgID.String()).  // Explicit string conversion
    WillReturnRows(rows)
```

#### Key Pattern 8: Soft Delete Filters

**GORM automatically adds `deleted_at IS NULL` to most queries:**

```go
// Repository code
db.First(&user, userID)

✅ CORRECT: Include deleted_at in WHERE
s.mock.ExpectQuery(`SELECT \* FROM "common"\."users" WHERE "users"\."id" = \$1 AND "users"\."deleted_at" IS NULL`).
    WithArgs(userID, 1).
    WillReturnRows(rows)

// For custom WHERE:
db.Where("organization_id = ?", orgID).Find(&devices)

✅ CORRECT: Parentheses + deleted_at
s.mock.ExpectQuery(`SELECT \* FROM "common"\."devices" WHERE \(organization_id = \$1 AND deleted_at IS NULL\)`).
    WithArgs(orgID).
    WillReturnRows(rows)
```

#### Key Pattern 9: JOIN Queries

**GORM generates complex JOIN queries with table aliases:**

```go
// Repository code
db.Table("organization_users ou").
    Joins("JOIN organizations o ON o.id = ou.organization_id").
    Where("ou.user_id = ?", userID).
    First(&result)

✅ CORRECT: Match exact JOIN structure with aliases
s.mock.ExpectQuery(
    `SELECT organization_users.organization_id FROM "common"."organization_users" ` +
    `JOIN common.organizations ON organizations.id = organization_users.organization_id ` +
    `WHERE \(organization_users.user_id = \$1 AND organization_users.is_active = \$2 ` +
    `AND organization_users.deleted_at IS NULL AND organizations.is_active = \$3 ` +
    `AND organizations.deleted_at IS NULL\)`,
).WithArgs(userID, true, true, 1).  // Don't forget LIMIT
WillReturnRows(rows)
```

#### Pattern 10: Count Queries

**Count queries include all WHERE clause filters:**

```go
// Repository code
db.Where("organization_id = ? AND is_active = ?", orgID, true).Count(&total)

✅ CORRECT: Include all filter arguments
s.mock.ExpectQuery(`SELECT count\(\*\) FROM "common"\."devices" WHERE \(organization_id = \$1 AND is_active = \$2 AND deleted_at IS NULL\)`).
    WithArgs(orgID, true).  // All filter values
    WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

❌ WRONG: Missing filter arguments
s.mock.ExpectQuery(`SELECT count\(\*\) FROM "common"\."devices"`).
    WithArgs(orgID).  // Missing is_active argument
    WillReturnRows(rows)
```

#### Pattern 11: Update Queries

**UPDATE queries include all WHERE conditions:**

```go
// Repository code
db.Model(&User{}).
    Where("id = ? AND deleted_at IS NULL", userID).
    Updates(map[string]interface{}{"name": newName})

✅ CORRECT:
s.mock.ExpectExec(`UPDATE "common"\."users" SET "name"=\$1,"updated_at"=\$2 WHERE \(id = \$3 AND deleted_at IS NULL\)`).
    WithArgs(newName, sqlmock.AnyArg(), userID).
    WillReturnResult(sqlmock.NewResult(0, 1))
```

#### Debugging sqlmock Mismatches

**When tests fail with "arguments do not match" or "remaining expectation":**

1. **Read the actual query from test output** - GORM logs the exact SQL it generates
2. **Count the arguments** - Error message shows expected vs actual count
3. **Check for these common issues:**
   - Missing LIMIT for First() queries
   - Missing Preload query expectations
   - Wrong argument count for pagination
   - IN clause not expanded to individual args
   - Missing WHERE parentheses in regex
   - Wrong schema name (common vs operation)

**Example debugging:**

```
Error: arguments do not match: expected 2, but got 3 arguments
Actual query: SELECT * FROM "devices" WHERE (org_id = $1 AND deleted_at IS NULL) ORDER BY created_at DESC LIMIT $3

Fix:
- Expected 2 args but got 3
- Missing argument is LIMIT value
- Add third argument to WithArgs()

s.mock.ExpectQuery(`SELECT \* FROM "devices"`).
    WithArgs(orgID, 10).  // Was: orgID only. Fixed: orgID + limit
    WillReturnRows(rows)
```

#### Test Isolation with sqlmock

**Use `SetupTest()` instead of `SetupSuite()` for test isolation:**

```go
✅ CORRECT: Fresh mock for each test
func (s *DeviceRepositoryTestSuite) SetupTest() {
    // Create new mock for each test
    var err error
    var sqlDB *sql.DB
    sqlDB, s.mock, err = sqlmock.New()
    s.Require().NoError(err)

    s.db, err = gorm.Open(postgres.New(postgres.Config{
        Conn: sqlDB,
    }), &gorm.Config{
        SkipDefaultTransaction: true,
    })
    s.Require().NoError(err)

    s.repo, err = NewDeviceRepository(s.db, s.cacheManager, false)
    s.Require().NoError(err)
}

❌ WRONG: Shared mock across all tests (causes expectation bleed)
func (s *DeviceRepositoryTestSuite) SetupSuite() {
    // Single mock shared by all tests
    // Leftover expectations pollute subsequent tests
    sqlDB, s.mock, _ = sqlmock.New()
    // ...
}
```

**Why this matters**: When tests share a sqlmock instance, unmet expectations from one test affect subsequent tests, causing mysterious failures.

#### Pattern 12: Mock Expectation Ordering

**CRITICAL**: Mock expectations must be in the EXACT order GORM executes them:

```go
// GORM executes queries in this order:
// 1. Main query (roles table)
// 2. Preload: role_permissions join table
// 3. Preload: permissions table
// 4. Preload: permission_dependencies (many-to-many)
// 5. Preload: permission_translations

✅ CORRECT: Expectations in execution order
s.mock.ExpectQuery(`SELECT \* FROM "common"\."roles"`).
    WithArgs(roleID, 1).
    WillReturnRows(roleRows)

s.mock.ExpectQuery(`SELECT \* FROM "common"\."role_permissions"`).
    WithArgs(roleID).
    WillReturnRows(rolePermRows)

s.mock.ExpectQuery(`SELECT \* FROM "common"\."permissions"`).
    WithArgs(permID).
    WillReturnRows(permRows)

❌ WRONG: Expectations out of order
s.mock.ExpectQuery(`SELECT \* FROM "common"\."permissions"`).  // Too early!
s.mock.ExpectQuery(`SELECT \* FROM "common"\."roles"`).        // Should be first
```

**How to determine order**:
1. Run the test and look at GORM's log output
2. Main query always executes first
3. Preloads execute in the order they're defined in repository code
4. Nested Preloads execute depth-first

#### Pattern 13: Error Path Testing

**When main query fails, don't mock Preload queries**:

```go
✅ CORRECT: Only mock the query that will fail
s.Run("NotFound", func() {
    // Main query returns error
    s.mock.ExpectQuery(`SELECT \* FROM "common"\."roles"`).
        WithArgs(roleID, 1).
        WillReturnError(gorm.ErrRecordNotFound)

    // NO Preload expectations - they won't execute!

    role, err := s.repo.GetRoleByID(ctx, roleID, locale)

    s.Error(err)
    s.Nil(role)
})

❌ WRONG: Mocking queries that won't execute
s.Run("NotFound", func() {
    s.mock.ExpectQuery(`SELECT \* FROM "common"\."roles"`).
        WillReturnError(gorm.ErrRecordNotFound)

    // These will never execute but are expected:
    s.mock.ExpectQuery(`SELECT \* FROM "common"\."permissions"`).  // ❌
    s.mock.ExpectQuery(`SELECT \* FROM "common"\."translations"`). // ❌
})
```

**Rule**: Only mock queries up to and including the one that returns an error.

#### Pattern 14: Test Assertion Safety

**Always check for nil before accessing struct fields**:

```go
✅ CORRECT: Check nil first
s.Run("EmptyResult", func() {
    // Mock setup...

    results, pagination, err := s.repo.GetData(ctx, id)

    s.NoError(err)
    s.NotNil(pagination, "Pagination should not be nil")  // Check first!
    s.Equal(0, pagination.Total)  // Safe to access now
    s.Empty(results)
})

❌ WRONG: Accessing without nil check
s.Run("EmptyResult", func() {
    results, pagination, err := s.repo.GetData(ctx, id)

    s.NoError(err)
    s.Equal(0, pagination.Total)  // ❌ PANIC if pagination is nil!
})
```

**Common panic locations**:
- `pagination.Total` - check `s.NotNil(pagination)` first
- `user.Email` - check `s.NotNil(user)` first
- `role.Name` - check `s.NotNil(role)` first

#### Pattern 15: Subtest Isolation

**Each subtest within s.Run() shares the same mock instance**:

```go
✅ CORRECT: Fresh expectations per subtest
func (s *TestSuite) TestGetUser() {
    userID := uuid.New()

    s.Run("Success", func() {
        // Set up expectations for this subtest
        s.mock.ExpectQuery(...).WillReturnRows(rows)

        user, err := s.repo.GetUser(ctx, userID)
        s.NoError(err)

        // Verify expectations were met
        s.NoError(s.mock.ExpectationsWereMet())
    })

    s.Run("NotFound", func() {
        // Set up NEW expectations for this subtest
        s.mock.ExpectQuery(...).WillReturnError(gorm.ErrRecordNotFound)

        user, err := s.repo.GetUser(ctx, userID)
        s.Error(err)

        // Verify expectations were met
        s.NoError(s.mock.ExpectationsWereMet())
    })
}

❌ WRONG: Leftover expectations from previous subtest
func (s *TestSuite) TestGetUser() {
    // Setting expectations outside s.Run()
    s.mock.ExpectQuery(...).WillReturnRows(rows)  // ❌ Wrong!

    s.Run("Success", func() {
        // Uses expectations from outside
    })
}
```

**Best practice**:
- Set up ALL expectations inside each `s.Run()`
- Call `s.mock.ExpectationsWereMet()` at end of each subtest
- Don't set up expectations outside of `s.Run()`

#### Pattern 16: Complex Preload Chains

**Understanding nested Preload execution order**:

```go
// Repository code:
db.Preload("Permissions.Dependencies.Translations").
   Preload("Permissions.Translations").
   First(&role, roleID)

// GORM executes in this order:
// 1. Main query: SELECT * FROM roles
// 2. First Preload: role_permissions (join table)
// 3. Permissions: SELECT * FROM permissions
// 4. Dependencies: permission_dependencies (join table)
// 5. Dependencies: SELECT * FROM permissions (for dependencies)
// 6. Translations for dependencies: SELECT * FROM permission_translations
// 7. Translations for permissions: SELECT * FROM permission_translations

✅ CORRECT: Mock all 7 queries in order
s.mock.ExpectQuery(`SELECT \* FROM "common"\."roles"`).WithArgs(roleID, 1)...
s.mock.ExpectQuery(`SELECT \* FROM "common"\."role_permissions"`).WithArgs(roleID)...
s.mock.ExpectQuery(`SELECT \* FROM "common"\."permissions" WHERE "permissions"\."id" IN`).WithArgs(perm1, perm2)...
s.mock.ExpectQuery(`SELECT \* FROM "common"\."permission_dependencies"`).WithArgs(perm1, perm2)...
s.mock.ExpectQuery(`SELECT \* FROM "common"\."permissions" WHERE "permissions"\."id" IN`).WithArgs(dep1)...
s.mock.ExpectQuery(`SELECT \* FROM "common"\."permission_translations" WHERE.*AND locale`).WithArgs(dep1, locale)...
s.mock.ExpectQuery(`SELECT \* FROM "common"\."permission_translations" WHERE.*AND locale`).WithArgs(perm1, perm2, locale)...
```

**Debugging tip**: Run test with verbose GORM logging to see exact query order.

#### Quick Reference Table

| Pattern | Example | Key Points |
|---------|---------|------------|
| First() queries | `WithArgs(id, 1)` | Always add LIMIT 1 as final arg |
| WHERE clauses | `WHERE (condition)` | Parentheses around custom WHERE |
| Preload | Mock main + related queries | Separate expectation for each Preload |
| IN clause | `WithArgs(id1, id2, id3)` | Each element is separate arg |
| Pagination | `WithArgs(filter, offset, limit)` | offset=0 is omitted |
| INSERT | timestamps first | created_at, deleted_by, deleted_at, then fields |
| UUIDs | Accept UUID or string | GORM converts, sqlmock handles both |
| Soft delete | `deleted_at IS NULL` | Added automatically to most queries |
| JOINs | Match exact structure | Include table aliases and all JOINs |
| Count | Include all filters | All WHERE args in count query |
| Test isolation | Use SetupTest() | Fresh mock per test prevents pollution |
| **Expectation order** | **Match GORM execution** | **Main query, then Preloads in order** |
| **Error paths** | **Mock only failing query** | **No Preload mocks after error** |
| **Nil checks** | **Check before accessing** | **Prevent panic on nil.Field** |
| **Subtest isolation** | **Expectations in s.Run()** | **Not outside** |
| **Complex Preloads** | **Depth-first order** | **Run test to see order** |

### Mock Location

**Repository mocks** are stored in `mocks/` subdirectory within each repository package:

```
pkg/database/repositories/users/
├── repo.go
├── repo_test.go
└── mocks/
    └── Repository.go           # Generated mock

pkg/database/repositories/roles/
├── repo.go
├── repo_test.go
└── mocks/
    └── Repository.go           # Generated mock
```

**Service tests** use mocks from other packages (no local `mocks/` folder):

```
pkg/services/access/
├── MODULE.go
├── SERVICE.go
├── service_test.go             # Imports mocks from repository packages
├── roles_test.go
└── permissions_test.go

# Services import mocks from:
# - pkg/database/repositories/*/mocks/Repository.go   (repository mocks)
# - pkg/database/transaction/mocks/Manager.go         (transaction manager mock)
# - pkg/common/jwt/mocks/Encoder.go                   (JWT encoder mock)
```

### Using Mocks in Tests

```go
import (
    "testing"
    "github.com/stretchr/testify/mock"
    "github.com/industrix-id/backend/pkg/database/repositories/mocks"
)

type UserServiceTestSuite struct {
    suite.Suite
    service      *userService
    mockUserRepo *mocks.UserRepository
    mockRoleRepo *mocks.RoleRepository
}

func (s *UserServiceTestSuite) SetupSuite() {
    // Initialize mocks
    s.mockUserRepo = new(mocks.UserRepository)
    s.mockRoleRepo = new(mocks.RoleRepository)

    // Create service with mocks
    s.service = &userService{
        userRepo: s.mockUserRepo,
        roleRepo: s.mockRoleRepo,
    }
}

func (s *UserServiceTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        // Arrange - setup mock expectations
        expectedUser := &schemas.User{
            ID:    uuid.New(),
            Email: "test@example.com",
        }

        s.mockUserRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*schemas.User")).
            Return(nil).
            Run(func(args mock.Arguments) {
                user := args.Get(1).(*schemas.User)
                user.ID = expectedUser.ID
            })

        // Act
        user, err := s.service.CreateUser(context.Background(), "test@example.com", "Test User")

        // Assert
        s.NoError(err)
        s.Equal(expectedUser.ID, user.ID)
        s.mockUserRepo.AssertExpectations(s.T())
    })
}
```

## Integration Test Setup Patterns

### Foreign Key Dependencies and Test Data Order

**CRITICAL**: Integration tests must create test data in dependency order to avoid foreign key violations.

#### Understanding Foreign Key Relationships

Many tables have foreign key dependencies that must be respected when creating test data:

```
Organizations (root entity)
    ↓
Sites (references organization_id)
    ↓
Device Types (references organization_id)
    ↓
Devices (references organization_id, site_id, device_type_id)
    ↓
Device Users (references device_id, user_id)
    ↓
FTM Process (references device_id)
```

#### Test Data Creation Pattern

**Always create parent records before child records:**

```go
func (s *DeviceRepositoryTestSuite) SetupSuite() {
    // 1. Create organization (root entity)
    s.testOrgID = uuid.New()
    testOrg := &schemas.Organization{
        ID:   s.testOrgID,
        Name: "Test Organization",
    }
    err = s.db.Create(testOrg).Error
    s.Require().NoError(err, "Failed to create test organization")

    // 2. Create site (depends on organization)
    s.testSiteID = uuid.New()
    testSite := &schemas.Site{
        ID:             s.testSiteID,
        Name:           "Test Site",
        OrganizationID: s.testOrgID,  // FK to organization
    }
    err = s.db.Create(testSite).Error
    s.Require().NoError(err, "Failed to create test site")

    // 3. Create device type (depends on organization)
    testDeviceType := &schemas.DeviceType{
        ID:             uuid.New(),
        Name:           "FTM v1",
        OrganizationID: s.testOrgID,  // FK to organization
    }
    err = s.db.Create(testDeviceType).Error
    s.Require().NoError(err, "Failed to create device type")

    // 4. Now safe to create devices (depends on org, site, device type)
}
```

#### Common Foreign Key Errors

**Error 1: Creating child before parent**
```
Error: insert or update on table "devices" violates foreign key constraint "fk_devices_organization"
Detail: Key (organization_id)=(xxx) is not present in table "organizations".
```

**Fix:** Create organization first, then device.

**Error 2: Deleting parent before children**
```
Error: update or delete on table "organizations" violates foreign key constraint "fk_devices_organization"
Detail: Key (id)=(xxx) is still referenced from table "devices".
```

**Fix:** Delete devices first, then organization (cleanup in reverse order).

#### Test Data Cleanup Pattern

**CRITICAL**: Clean up test data in REVERSE order of creation to avoid FK violations.

```go
func (s *DeviceRepositoryTestSuite) TearDownSuite() {
    // Clean up in REVERSE order (children first, parents last)

    // 5. Delete FTM process data (deepest dependency)
    s.db.Unscoped().Where("device_id IN (?)",
        s.db.Model(&schemas.Device{}).
            Select("id").
            Where("organization_id = ?", s.testOrgID),
    ).Delete(&schemas.FTMProcess{})

    // 4. Delete device users
    s.db.Unscoped().Where("device_id IN (?)",
        s.db.Model(&schemas.Device{}).
            Select("id").
            Where("organization_id = ?", s.testOrgID),
    ).Delete(&schemas.DeviceUser{})

    // 3. Delete devices
    s.db.Unscoped().Where("organization_id = ?", s.testOrgID).
        Delete(&schemas.Device{})

    // 2. Delete sites
    s.db.Unscoped().Where("organization_id = ?", s.testOrgID).
        Delete(&schemas.Site{})

    // 1. Delete organization (root entity, deleted last)
    s.db.Unscoped().Where("id = ?", s.testOrgID).
        Delete(&schemas.Organization{})

    // Close database connection
    sqlDB, err := s.db.DB()
    if err == nil {
        _ = sqlDB.Close()
    }
}
```

#### Best Practices for Integration Test Setup

**SetupSuite vs SetupTest:**
```go
// Use SetupSuite for shared resources (DB connection, test org)
func (s *IntegrationTestSuite) SetupSuite() {
    // Connect to database (expensive, do once)
    // Create test organization (shared across tests)
    // Create common lookup data (device types, etc.)
}

// Use SetupTest for test-specific isolation
func (s *IntegrationTestSuite) SetupTest() {
    // Clean up any leftover test data from previous tests
    s.db.Unscoped().Where("name LIKE ?", "test%").Delete(&schemas.Device{})
}

// Use TearDownTest for per-test cleanup
func (s *IntegrationTestSuite) TearDownTest() {
    // Clean up test-specific data immediately after each test
    s.db.Unscoped().Where("name LIKE ?", "test%").Delete(&schemas.Device{})
}

// Use TearDownSuite for final cleanup
func (s *IntegrationTestSuite) TearDownSuite() {
    // Delete ALL test data in reverse dependency order
    // Close database connection
}
```

#### Cleanup Helpers for Complex Dependencies

```go
// Helper function to clean up organization and all related data
func (s *TestSuite) cleanupOrganization(orgID uuid.UUID) {
    // Delete in reverse dependency order
    s.db.Exec("DELETE FROM ftm_process WHERE device_id IN (SELECT id FROM devices WHERE organization_id = ?)", orgID)
    s.db.Exec("DELETE FROM device_users WHERE device_id IN (SELECT id FROM devices WHERE organization_id = ?)", orgID)
    s.db.Exec("DELETE FROM devices WHERE organization_id = ?", orgID)
    s.db.Exec("DELETE FROM organization_sites WHERE organization_id = ?", orgID)
    s.db.Exec("DELETE FROM sites WHERE organization_id = ?", orgID)
    s.db.Exec("DELETE FROM organization_users WHERE organization_id = ?", orgID)
    s.db.Exec("DELETE FROM organizations WHERE id = ?", orgID)
}
```

#### Complete Integration Test Example

```go
type SiteRepositoryIntegrationTestSuite struct {
    suite.Suite
    repo         SiteRepository
    db           *gorm.DB
    cacheManager *caches.CacheManager
    testOrgID    uuid.UUID
}

func (s *SiteRepositoryIntegrationTestSuite) SetupSuite() {
    // 1. Connect to database
    dsn := fmt.Sprintf("host=%s port=%d...", ...)
    s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    s.Require().NoError(err, "Failed to connect to test database")

    // 2. Run migrations
    err = s.db.AutoMigrate(&schemas.Site{}, &schemas.Organization{})
    s.Require().NoError(err, "Failed to run migrations")

    // 3. Create test organization (parent entity)
    s.testOrgID = uuid.New()
    testOrg := &schemas.Organization{
        ID:   s.testOrgID,
        Name: "Test Organization",
    }
    err = s.db.Create(testOrg).Error
    s.Require().NoError(err, "Failed to create test organization")

    // 4. Initialize repository
    s.repo, err = NewSiteRepository(s.db, s.cacheManager, false)
    s.Require().NoError(err, "Failed to create SiteRepository")
}

func (s *SiteRepositoryIntegrationTestSuite) SetupTest() {
    // Clean up leftover test data before each test
    s.db.Unscoped().Where("name LIKE ?", "test%").Delete(&schemas.Site{})
}

func (s *SiteRepositoryIntegrationTestSuite) TearDownTest() {
    // Clean up test-specific data after each test
    s.db.Unscoped().Where("name LIKE ?", "test%").Delete(&schemas.Site{})
}

func (s *SiteRepositoryIntegrationTestSuite) TearDownSuite() {
    // Clean up in reverse order (children first)
    s.db.Unscoped().Where("name LIKE ?", "test%").Delete(&schemas.Site{})
    s.db.Unscoped().Where("organization_id = ?", s.testOrgID).Delete(&schemas.OrganizationSite{})
    s.db.Unscoped().Where("id = ?", s.testOrgID).Delete(&schemas.Organization{})

    // Close database connection
    sqlDB, err := s.db.DB()
    if err == nil {
        _ = sqlDB.Close()
    }
}
```

## Database Testing Modes

We support **three testing modes** for database interactions, with **Redis integration required for all integration tests** that use caching.

### Redis Integration in Integration Tests

**CRITICAL**: All repositories that use `BaseRepository` (which provides caching) **MUST test Redis integration**, not just PostgreSQL.

**Why This Matters**:
- 26+ repositories use Redis caching extensively
- Cache invalidation bugs can cause serious production issues
- TTL expiration, concurrent access, and cache key conflicts are only caught with real Redis
- Mock tests don't catch Redis-specific issues (key format errors, connection failures, eviction policies)

**Required for Integration Tests**:
```go
//go:build integration || integration_all
// +build integration integration_all

type UserRepositoryIntegrationTestSuite struct {
    suite.Suite
    repo         UserRepository
    db           *gorm.DB
    redisClient  *redis.Client      // ✅ REQUIRED: Real Redis client
    cacheManager *caches.CacheManager
    testOrgID    uuid.UUID
}

func (s *UserRepositoryIntegrationTestSuite) SetupSuite() {
    // 1. Connect to PostgreSQL (existing pattern)
    dsn := fmt.Sprintf("host=%s port=%d...", ...)
    s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    s.Require().NoError(err)

    // 2. Connect to Redis (NEW REQUIREMENT)
    redisHost := os.Getenv("TEST_REDIS_HOST")
    if redisHost == "" {
        redisHost = "localhost"
    }

    redisPort, _ := strconv.Atoi(getEnvOrDefault("TEST_REDIS_PORT", "6379"))
    redisPassword := os.Getenv("TEST_REDIS_PASSWORD")
    redisDB, _ := strconv.Atoi(getEnvOrDefault("TEST_REDIS_DB", "0"))

    s.redisClient = redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
        Password: redisPassword,
        DB:       redisDB,
    })

    // Test Redis connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    err = s.redisClient.Ping(ctx).Err()
    s.Require().NoError(err, "Failed to connect to Redis. Set TEST_REDIS_HOST and TEST_REDIS_PORT.")

    // 3. Initialize cache manager with REAL Redis
    logger, _ := zap.NewDevelopment()
    s.cacheManager = caches.NewCacheManager(s.redisClient, true, logger.Sugar())

    // 4. Initialize repository with caching ENABLED
    s.repo, err = NewUserRepository(s.db, s.cacheManager, true)  // true = caching enabled
    s.Require().NoError(err)
}

func (s *UserRepositoryIntegrationTestSuite) TearDownSuite() {
    // Clean up Redis keys
    ctx := context.Background()
    keys, _ := s.redisClient.Keys(ctx, "user:*").Result()
    if len(keys) > 0 {
        s.redisClient.Del(ctx, keys...)
    }

    // Close Redis connection
    if s.redisClient != nil {
        _ = s.redisClient.Close()
    }

    // Clean up database (existing pattern)
    // ...
}
```

**Environment Variables for Integration Tests**:
```bash
# PostgreSQL (existing)
TEST_DB_HOST=localhost
TEST_DB_PORT=5432
TEST_DB_USER=test_user
TEST_DB_PASSWORD=test_password
TEST_DB_NAME=industrix_test

# Redis (NEW REQUIREMENT)
TEST_REDIS_HOST=localhost      # Use "redis" in CI/Docker
TEST_REDIS_PORT=6379
TEST_REDIS_PASSWORD=""          # Usually empty for test Redis
TEST_REDIS_DB=0                 # Use DB 0 for tests
```

**What to Test with Real Redis**:
1. **Cache Hit/Miss**: Verify data is cached and retrieved correctly
2. **Cache Invalidation**: Verify cache is properly cleared after updates/deletes
3. **TTL Expiration**: Verify cache keys expire as expected
4. **Concurrent Access**: Verify multiple operations don't cause race conditions
5. **Cache Key Format**: Verify cache keys are correctly formatted and unique

**Example Integration Test with Redis**:
```go
func (s *UserRepositoryIntegrationTestSuite) TestGetUserByID_WithCaching() {
    s.Run("CacheMiss_ThenCacheHit", func() {
        // Create test user in database
        user := &schemas.User{
            ID:    uuid.New(),
            Email: "test@example.com",
            Name:  "Test User",
        }
        err := s.db.Create(user).Error
        s.Require().NoError(err)
        defer s.db.Unscoped().Delete(user)

        // First call - cache miss, should hit database
        result1, err := s.repo.GetUserByID(context.Background(), user.ID)
        s.NoError(err)
        s.Equal(user.Email, result1.Email)

        // Verify data is now in cache
        cacheKey := fmt.Sprintf("user:%s", user.ID.String())
        exists, _ := s.redisClient.Exists(context.Background(), cacheKey).Result()
        s.Equal(int64(1), exists, "User should be cached after first retrieval")

        // Second call - cache hit, should NOT hit database
        result2, err := s.repo.GetUserByID(context.Background(), user.ID)
        s.NoError(err)
        s.Equal(user.Email, result2.Email)

        // Both results should be identical
        s.Equal(result1.ID, result2.ID)
        s.Equal(result1.Email, result2.Email)
    })

    s.Run("CacheInvalidation_AfterUpdate", func() {
        // Create and cache user
        user := &schemas.User{
            ID:    uuid.New(),
            Email: "original@example.com",
            Name:  "Original Name",
        }
        s.db.Create(user)
        defer s.db.Unscoped().Delete(user)

        // Cache the user
        _, _ = s.repo.GetUserByID(context.Background(), user.ID)

        // Update user (should invalidate cache)
        err := s.repo.UpdateUser(context.Background(), user.ID, "updated@example.com", "Updated Name")
        s.NoError(err)

        // Verify cache was invalidated
        cacheKey := fmt.Sprintf("user:%s", user.ID.String())
        exists, _ := s.redisClient.Exists(context.Background(), cacheKey).Result()
        s.Equal(int64(0), exists, "Cache should be invalidated after update")

        // Next retrieval should get updated data from database
        result, err := s.repo.GetUserByID(context.Background(), user.ID)
        s.NoError(err)
        s.Equal("updated@example.com", result.Email)
    })
}
```

**Reference Implementation**:
See `pkg/database/repositories/ftm_live_flow_rate_integration_test.go` for a complete example of Redis-only integration testing.

### Mode 1: Mock Only (Default)

Use sqlmock and redismock for fast, isolated tests without external dependencies.

**Build Tag Pattern**: Unit tests exclude only the `integration` tag, allowing them to run with `integration_all`.

```go
//go:build !integration
// +build !integration

func (s *UserRepositoryTestSuite) SetupSuite() {
    // Setup sqlmock
    var sqlDB *sql.DB
    sqlDB, s.mock, _ = sqlmock.New()

    s.db, _ = gorm.Open(postgres.New(postgres.Config{
        Conn: sqlDB,
    }), &gorm.Config{})

    s.repo, _ = NewUserRepository(s.db, nil, false)
}

func (s *UserRepositoryTestSuite) TestGetUserByID() {
    s.Run("Success", func() {
        userID := uuid.New()

        // Mock database response
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).
                AddRow(userID, "test@example.com"))

        user, err := s.repo.GetUserByID(context.Background(), userID)

        s.NoError(err)
        s.Equal(userID, user.ID)
    })
}
```

**Run**: `go test ./...` or `make test`

### Mode 2: Test Database

Use real PostgreSQL database for integration testing.

```go
//go:build integration || integration_all
// +build integration integration_all

func (s *UserRepositoryTestSuite) SetupSuite() {
    // Connect to test database
    dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("TEST_DB_HOST"),
        5432,
        os.Getenv("TEST_DB_USER"),
        os.Getenv("TEST_DB_PASSWORD"),
        os.Getenv("TEST_DB_NAME"),
    )

    var err error
    s.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    s.Require().NoError(err)

    // Run migrations
    s.db.AutoMigrate(&schemas.User{})

    s.repo, err = NewUserRepository(s.db, nil, false)
    s.Require().NoError(err)
}

func (s *UserRepositoryTestSuite) TearDownSuite() {
    // Clean up test data
    s.db.Exec("DELETE FROM users WHERE email LIKE 'test%'")
}

func (s *UserRepositoryTestSuite) TestGetUserByID() {
    s.Run("Success", func() {
        // Insert test data
        user := &schemas.User{
            ID:    uuid.New(),
            Email: "test@example.com",
            Name:  "Test User",
        }
        s.db.Create(user)
        defer s.db.Delete(user) // Cleanup

        // Test against real database
        result, err := s.repo.GetUserByID(context.Background(), user.ID)

        s.NoError(err)
        s.Equal(user.ID, result.ID)
        s.Equal(user.Email, result.Email)
    })
}
```

**Run**: `go test -tags=integration ./...`

#### CRITICAL: GORM Zero-Value Issue with Boolean Fields

**GORM skips boolean fields set to `false` during `Create()` operations, causing database defaults to be used instead.**

This is a fundamental GORM behavior when working with zero values in Go:
- Go's zero value for `bool` is `false`
- GORM treats `false` as a zero value and skips it during `Create()`
- Database default value (often `true`) is used instead
- Result: Test data doesn't match your intentions

**Problem Example:**
```go
❌ WRONG: This will NOT create an inactive device
inactiveDevice := &schemas.Device{
    ID:             uuid.New(),
    Name:           "Inactive Device",
    OrganizationID: orgID,
    IsActive:       false,  // ❌ GORM IGNORES THIS!
}
err := s.db.Create(inactiveDevice).Error  // Creates with is_active=true (DB default)

// Later in test:
// Expected: Only active devices counted
// Actual: Both devices counted (inactive device has is_active=true!)
```

**The Issue:**
1. Database schema: `is_active boolean NOT NULL DEFAULT true`
2. GORM behavior: Skips `false` value during `Create()`
3. Database uses default: `is_active = true`
4. Test fails: "Inactive" device is actually active!

**Solution Patterns:**

**Pattern A: Create + Update (Recommended)**
```go
✅ CORRECT: Create first, then update to false
inactiveDevice := &schemas.Device{
    ID:             uuid.New(),
    Name:           "Inactive Device",
    OrganizationID: orgID,
    IsActive:       false,  // This gets ignored by Create()
}

// Step 1: Create with database default (true)
err := s.db.Create(inactiveDevice).Error
s.Require().NoError(err)

// Step 2: Explicitly update to false using Model + Update
err = s.db.Model(&schemas.Device{}).
    Where("id = ?", inactiveDevice.ID).
    Update("is_active", false).Error
s.Require().NoError(err)

defer s.db.Unscoped().Delete(inactiveDevice)

// Now inactiveDevice.IsActive is truly false in database
```

**Pattern B: Using Select() (Alternative)**
```go
✅ ALTERNATIVE: Force GORM to include the field
inactiveDevice := &schemas.Device{
    ID:             uuid.New(),
    Name:           "Inactive Device",
    OrganizationID: orgID,
    IsActive:       false,
}

// Explicitly tell GORM to include is_active field
err := s.db.Select("ID", "Name", "OrganizationID", "IsActive").
    Create(inactiveDevice).Error
s.Require().NoError(err)

defer s.db.Unscoped().Delete(inactiveDevice)

// Note: This is less reliable - Create + Update is preferred
```

**When This Applies:**
- ✅ Boolean fields: `is_active`, `is_verified`, `is_deleted`, etc.
- ✅ Any field where Go zero value matches the type's zero value
- ✅ Integration tests creating test data with `false` values
- ⚠️ NOT needed for sqlmock unit tests (mocks don't have this behavior)

**Why Create + Update Pattern is Preferred:**
1. **Explicit and Clear**: Clearly shows the workaround and why it exists
2. **Reliable**: Always works regardless of GORM version
3. **Self-Documenting**: Code comments explain the GORM limitation
4. **Consistent**: Works for all boolean fields and zero-value scenarios

**Complete Integration Test Example:**
```go
func (s *DeviceRepositoryTestSuite) TestCountActiveFuelTankDevicesByOrganization_Integration() {
    s.Run("OnlyCountsActiveDevices", func() {
        orgID := uuid.New()

        // Create active device (no issue - true is not zero value)
        activeDevice := &schemas.Device{
            ID:             uuid.New(),
            Name:           "Active FTM",
            DeviceType:     utils.ToPointer(string(constants.DeviceTypeFuelTankMonitoringV1)),
            OrganizationID: orgID,
            IsActive:       true,  // Works fine
        }
        err := s.db.Create(activeDevice).Error
        s.Require().NoError(err)
        defer s.db.Unscoped().Delete(activeDevice)

        // Create inactive device (REQUIRES WORKAROUND)
        inactiveDevice := &schemas.Device{
            ID:             uuid.New(),
            Name:           "Inactive FTM",
            DeviceType:     utils.ToPointer(string(constants.DeviceTypeFuelTankMonitoringV1)),
            OrganizationID: orgID,
            IsActive:       false,  // This gets ignored by GORM Create()
        }
        err = s.db.Create(inactiveDevice).Error
        s.Require().NoError(err)

        // CRITICAL: Explicitly set is_active to false after creation
        // (workaround for GORM zero-value issue)
        err = s.db.Model(&schemas.Device{}).
            Where("id = ?", inactiveDevice.ID).
            Update("is_active", false).Error
        s.Require().NoError(err)
        defer s.db.Unscoped().Delete(inactiveDevice)

        // Now test - should only count active device
        count, err := s.repo.CountActiveFuelTankDevicesByOrganization(context.Background(), orgID)

        s.NoError(err)
        s.Equal(int64(1), count, "Should only count active device")
    })
}
```

**Common Mistakes to Avoid:**
```go
❌ WRONG: Trying Select("*") or Select() with all fields
err := s.db.Select("*").Create(device).Error  // Still skips false values

❌ WRONG: Using Updates() instead of Update()
err := s.db.Model(&Device{}).
    Where("id = ?", id).
    Updates(map[string]interface{}{"is_active": false}).Error  // Use Update(), not Updates()

❌ WRONG: Assuming GORM v2 fixed this
// This is expected GORM behavior for zero values, not a bug
```

**Best Practices:**
1. **Always use Create + Update pattern for boolean fields set to `false`**
2. **Add clear comments explaining the GORM zero-value workaround**
3. **Test that your test data matches expectations** (add debug logging if needed)
4. **Use defer for cleanup immediately after creation** to ensure cleanup happens
5. **Prefer `Update()` over `Updates()` for single field updates**

### Mode 3: Both Modes (Unit + Integration)

Run both mock and integration tests together using the `integration_all` tag.

**How it Works**:
- **Unit test files** use `//go:build !integration` (exclude only integration tag)
- **Integration test files** use `//go:build integration || integration_all` (include both tags)
- **Result**: `integration_all` tag runs BOTH test types ✅

**Run**: `go test -tags=integration_all ./...` or `make test-all`

### Configuration

Set environment variables for test database:

```bash
# .env.test
TEST_DB_HOST=localhost
TEST_DB_PORT=5432
TEST_DB_USER=test_user
TEST_DB_PASSWORD=test_password
TEST_DB_NAME=industrix_test
```

## Test Patterns

### Arrange-Act-Assert (AAA) Pattern

Every test should follow the AAA structure:

```go
func (s *UserRepositoryTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        // Arrange - Setup test data and expectations
        user := &schemas.User{
            Email: "test@example.com",
            Name:  "Test User",
        }

        s.mock.ExpectBegin()
        s.mock.ExpectQuery(`INSERT INTO "users"`).
            WithArgs(sqlmock.AnyArg(), user.Email, user.Name, sqlmock.AnyArg(), sqlmock.AnyArg()).
            WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
        s.mock.ExpectCommit()

        // Act - Execute the function under test
        err := s.repo.CreateUser(context.Background(), user)

        // Assert - Verify the results
        s.NoError(err)
        s.NotEqual(uuid.Nil, user.ID)
    })
}
```

### Table-Driven Tests with Inline Cases

For testing multiple scenarios with similar structure:

```go
func (s *ValidationTestSuite) TestValidateEmail() {
    testCases := []struct {
        name        string
        email       string
        shouldError bool
        errorMsg    string
    }{
        {"Valid email", "user@example.com", false, ""},
        {"Empty email", "", true, "email is required"},
        {"Invalid format", "invalid-email", true, "invalid email format"},
        {"Missing domain", "user@", true, "invalid email format"},
        {"Too long", strings.Repeat("a", 256) + "@example.com", true, "email too long"},
    }

    for _, tc := range testCases {
        s.Run(tc.name, func() {
            err := validateEmail(tc.email)

            if tc.shouldError {
                s.Error(err)
                s.Contains(err.Error(), tc.errorMsg)
            } else {
                s.NoError(err)
            }
        })
    }
}
```

### Comprehensive Error Testing

Test all error paths, not just the happy path:

```go
func (s *UserRepositoryTestSuite) TestGetUserByID() {
    userID := uuid.New()

    s.Run("Success", func() {
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name"}).
                AddRow(userID, "user@example.com", "Test User"))

        user, err := s.repo.GetUserByID(context.Background(), userID)

        s.NoError(err)
        s.NotNil(user)
        s.Equal(userID, user.ID)
    })

    s.Run("NotFound", func() {
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnError(gorm.ErrRecordNotFound)

        user, err := s.repo.GetUserByID(context.Background(), userID)

        s.Error(err)
        s.Nil(user)
        s.Contains(err.Error(), "not found")
    })

    s.Run("DatabaseError", func() {
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnError(errors.New("connection lost"))

        user, err := s.repo.GetUserByID(context.Background(), userID)

        s.Error(err)
        s.Nil(user)
    })

    s.Run("ContextTimeout", func() {
        ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
        defer cancel()

        time.Sleep(2 * time.Millisecond)

        user, err := s.repo.GetUserByID(ctx, userID)

        s.Error(err)
        s.Nil(user)
        s.Contains(err.Error(), "context deadline exceeded")
    })
}
```

### Edge Cases and Boundary Values

Always test edge cases:

```go
func (s *UserServiceTestSuite) TestCreateUser() {
    s.Run("EmptyEmail", func() {
        user, err := s.service.CreateUser(context.Background(), "", "Test User")

        s.Error(err)
        s.Nil(user)
        s.Contains(err.Error(), "email")
    })

    s.Run("EmptyName", func() {
        user, err := s.service.CreateUser(context.Background(), "test@example.com", "")

        s.Error(err)
        s.Nil(user)
        s.Contains(err.Error(), "name")
    })

    s.Run("NilContext", func() {
        user, err := s.service.CreateUser(nil, "test@example.com", "Test User")

        s.Error(err)
        s.Nil(user)
    })

    s.Run("VeryLongEmail", func() {
        longEmail := strings.Repeat("a", 256) + "@example.com"
        user, err := s.service.CreateUser(context.Background(), longEmail, "Test User")

        s.Error(err)
        s.Nil(user)
    })
}
```

### Mock Setup Patterns

**Successful Operation Mock:**

```go
s.mock.ExpectBegin()
s.mock.ExpectQuery(`INSERT INTO "users"`).
    WithArgs(sqlmock.AnyArg()).
    WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
s.mock.ExpectCommit()
```

**Failed Operation Mock:**

```go
s.mock.ExpectBegin()
s.mock.ExpectQuery(`INSERT INTO "users"`).
    WithArgs(sqlmock.AnyArg()).
    WillReturnError(errors.New("duplicate key"))
s.mock.ExpectRollback()
```

**Query with Multiple Rows:**

```go
rows := sqlmock.NewRows([]string{"id", "email", "name"}).
    AddRow(uuid.New(), "user1@example.com", "User 1").
    AddRow(uuid.New(), "user2@example.com", "User 2").
    AddRow(uuid.New(), "user3@example.com", "User 3")

s.mock.ExpectQuery("SELECT .* FROM \"users\"").
    WillReturnRows(rows)
```

## Assertions

### Suite Assertion Methods

Use suite methods for cleaner assertions:

```go
// Basic assertions
s.NoError(err)              // Assert no error occurred
s.Error(err)                // Assert error occurred
s.Nil(value)                // Assert value is nil
s.NotNil(value)             // Assert value is not nil
s.True(condition)           // Assert condition is true
s.False(condition)          // Assert condition is false

// Equality assertions
s.Equal(expected, actual)   // Assert values are equal
s.NotEqual(expected, actual) // Assert values are not equal
s.Empty(value)              // Assert value is empty
s.NotEmpty(value)           // Assert value is not empty

// Collection assertions
s.Len(slice, expectedLength)  // Assert slice/array length
s.Contains(haystack, needle)  // Assert contains substring/element
s.NotContains(haystack, needle) // Assert does not contain

// Type assertions
s.IsType(&User{}, result)   // Assert result is of specific type
s.Implements((*Interface)(nil), instance) // Assert implements interface

// Comparison assertions
s.Greater(actual, expected)      // Assert actual > expected
s.GreaterOrEqual(actual, expected) // Assert actual >= expected
s.Less(actual, expected)         // Assert actual < expected
s.LessOrEqual(actual, expected)  // Assert actual <= expected
```

### Common Assertion Patterns

**Error Assertions:**

```go
// Assert specific error
s.ErrorIs(err, gorm.ErrRecordNotFound)

// Assert error contains message
s.Error(err)
s.Contains(err.Error(), "not found")

// Assert custom error type
var customErr *common.CustomError
s.ErrorAs(err, &customErr)
s.Equal(errorcodes.NotFound, customErr.Code)
```

**Struct Assertions:**

```go
// Assert struct fields
s.Equal(expectedUser.ID, actualUser.ID)
s.Equal(expectedUser.Email, actualUser.Email)
s.Equal(expectedUser.Name, actualUser.Name)

// Assert partial struct match
s.Equal(expected.ID, actual.ID)
s.NotEmpty(actual.CreatedAt)
s.NotEmpty(actual.UpdatedAt)
```

**Collection Assertions:**

```go
// Assert slice length
s.Len(users, 3)

// Assert slice contains element
s.Contains(userIDs, targetID)

// Assert all elements match condition
for _, user := range users {
    s.NotEmpty(user.Email)
    s.True(user.IsActive)
}
```

**Time Assertions:**

```go
// Assert time approximately equal (within tolerance)
s.WithinDuration(expectedTime, actualTime, time.Second)

// Assert time is recent
s.True(time.Since(user.CreatedAt) < time.Minute)

// Assert time ordering
s.True(user.UpdatedAt.After(user.CreatedAt))
```

### Require vs Assert

Use `s.Require()` for critical assertions that should stop test execution:

```go
func (s *UserRepositoryTestSuite) TestGetUserByID() {
    userID := uuid.New()

    s.Run("Success", func() {
        s.mock.ExpectQuery("SELECT .* FROM \"users\"").
            WithArgs(userID).
            WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).
                AddRow(userID, "user@example.com"))

        user, err := s.repo.GetUserByID(context.Background(), userID)

        // Use Require for critical assertions
        s.Require().NoError(err, "GetUserByID should not return error")
        s.Require().NotNil(user, "User should not be nil")

        // Use regular assertions for field checks
        s.Equal(userID, user.ID)
        s.Equal("user@example.com", user.Email)
    })
}
```

**When to use Require:**
- Setup validation in `SetupSuite()` or `SetupTest()`
- Critical preconditions (e.g., no error, not nil)
- Assertions where continuing would cause panic

**When to use Assert:**
- Field validation
- Multiple independent checks
- Non-critical assertions

## Test Helpers

### Fixture Builders

Create test data builders for common structures:

```go
// testhelpers/builders.go
package testhelpers

import (
    "github.com/google/uuid"
    "github.com/industrix-id/backend/pkg/database/schemas"
)

// UserBuilder provides fluent API for building test users
type UserBuilder struct {
    user *schemas.User
}

func NewUserBuilder() *UserBuilder {
    return &UserBuilder{
        user: &schemas.User{
            ID:       uuid.New(),
            Email:    "test@example.com",
            Name:     "Test User",
            IsActive: true,
        },
    }
}

func (b *UserBuilder) WithID(id uuid.UUID) *UserBuilder {
    b.user.ID = id
    return b
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
    b.user.Email = email
    return b
}

func (b *UserBuilder) WithName(name string) *UserBuilder {
    b.user.Name = name
    return b
}

func (b *UserBuilder) Inactive() *UserBuilder {
    b.user.IsActive = false
    return b
}

func (b *UserBuilder) Build() *schemas.User {
    return b.user
}

// Usage in tests
user := testhelpers.NewUserBuilder().
    WithEmail("custom@example.com").
    WithName("Custom User").
    Build()
```

### Assertion Helpers

Create custom assertion helpers for common validations:

```go
// testhelpers/assertions.go
package testhelpers

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/industrix-id/backend/pkg/database/schemas"
)

// AssertValidUser checks if user has all required fields
func AssertValidUser(t *testing.T, user *schemas.User) {
    assert.NotEqual(t, uuid.Nil, user.ID, "User ID should not be nil")
    assert.NotEmpty(t, user.Email, "User email should not be empty")
    assert.NotEmpty(t, user.Name, "User name should not be empty")
    assert.NotEmpty(t, user.CreatedAt, "CreatedAt should be set")
    assert.NotEmpty(t, user.UpdatedAt, "UpdatedAt should be set")
}

// AssertValidRole checks if role has all required fields
func AssertValidRole(t *testing.T, role *schemas.Role) {
    assert.NotEqual(t, uuid.Nil, role.ID, "Role ID should not be nil")
    assert.NotEmpty(t, role.Name, "Role name should not be empty")
    assert.NotEmpty(t, role.OrganizationID, "Organization ID should not be nil")
    assert.NotEmpty(t, role.CreatedAt, "CreatedAt should be set")
}

// AssertPaginationInfo checks if pagination info is valid
func AssertPaginationInfo(t *testing.T, page *common.PaginationInfo, expectedTotal int) {
    assert.NotNil(t, page, "Pagination info should not be nil")
    assert.Equal(t, expectedTotal, page.Total, "Total should match")
    assert.Greater(t, page.Page, 0, "Page should be greater than 0")
    assert.Greater(t, page.Limit, 0, "Limit should be greater than 0")
}

// Usage in tests
func (s *UserRepositoryTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        user := testhelpers.NewUserBuilder().Build()

        err := s.repo.CreateUser(context.Background(), user)
        s.NoError(err)

        testhelpers.AssertValidUser(s.T(), user)
    })
}
```

### Mock Setup Helpers

Create helpers for common mock setups:

```go
// testhelpers/mocks.go
package testhelpers

import (
    "github.com/DATA-DOG/go-sqlmock"
    "github.com/google/uuid"
)

// ExpectUserQuery sets up mock for user query
func ExpectUserQuery(mock sqlmock.Sqlmock, userID uuid.UUID, email, name string) {
    rows := sqlmock.NewRows([]string{"id", "email", "name", "is_active", "created_at", "updated_at"}).
        AddRow(userID, email, name, true, time.Now(), time.Now())

    mock.ExpectQuery(`SELECT .* FROM "users"`).
        WithArgs(userID).
        WillReturnRows(rows)
}

// ExpectUserCreate sets up mock for user creation
func ExpectUserCreate(mock sqlmock.Sqlmock, user *schemas.User) {
    mock.ExpectBegin()
    mock.ExpectQuery(`INSERT INTO "users"`).
        WithArgs(sqlmock.AnyArg(), user.Email, user.Name, true, sqlmock.AnyArg(), sqlmock.AnyArg()).
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(user.ID))
    mock.ExpectCommit()
}

// ExpectRoleQuery sets up mock for role query with permissions
func ExpectRoleQueryWithPermissions(mock sqlmock.Sqlmock, roleID uuid.UUID, permissionCount int) {
    roleRows := sqlmock.NewRows([]string{"id", "name", "organization_id", "is_active"}).
        AddRow(roleID, "Test Role", uuid.New(), true)

    mock.ExpectQuery(`SELECT .* FROM "roles"`).
        WithArgs(roleID).
        WillReturnRows(roleRows)

    permRows := sqlmock.NewRows([]string{"id", "name", "resource", "action"})
    for i := 0; i < permissionCount; i++ {
        permRows.AddRow(uuid.New(), "Permission", "resource", "action")
    }

    mock.ExpectQuery(`SELECT .* FROM "permissions"`).
        WillReturnRows(permRows)
}

// Usage in tests
func (s *UserRepositoryTestSuite) TestGetUserByID() {
    s.Run("Success", func() {
        userID := uuid.New()
        testhelpers.ExpectUserQuery(s.mock, userID, "test@example.com", "Test User")

        user, err := s.repo.GetUserByID(context.Background(), userID)

        s.NoError(err)
        testhelpers.AssertValidUser(s.T(), user)
    })
}
```

### Context Helpers

Create helpers for common context setups:

```go
// testhelpers/context.go
package testhelpers

import (
    "context"
    "time"
)

// NewTestContext creates a context with reasonable timeout
func NewTestContext() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), 5*time.Second)
}

// NewShortContext creates a context with short timeout for testing timeouts
func NewShortContext() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), 1*time.Millisecond)
}

// Usage in tests
func (s *UserServiceTestSuite) TestCreateUser_ContextTimeout() {
    s.Run("ContextTimeout", func() {
        ctx, cancel := testhelpers.NewShortContext()
        defer cancel()

        time.Sleep(2 * time.Millisecond)

        user, err := s.service.CreateUser(ctx, "test@example.com", "Test User")

        s.Error(err)
        s.Nil(user)
        s.Contains(err.Error(), "context deadline exceeded")
    })
}
```

## Documentation Standards

### Test Comments

Document test purpose and expectations:

```go
// TestCreateUser tests user creation functionality.
// It covers successful creation, validation errors, and database errors.
func (s *UserRepositoryTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        // Test that a valid user is created successfully
        user := &schemas.User{
            Email: "test@example.com",
            Name:  "Test User",
        }

        err := s.repo.CreateUser(context.Background(), user)

        s.NoError(err)
        s.NotEqual(uuid.Nil, user.ID)
    })

    s.Run("EmptyEmail", func() {
        // Test that empty email returns validation error
        user := &schemas.User{
            Email: "",
            Name:  "Test User",
        }

        err := s.repo.CreateUser(context.Background(), user)

        s.Error(err)
        s.Contains(err.Error(), "email")
    })
}
```

### Inline Comments

Use inline comments for complex test logic:

```go
func (s *RoleRepositoryTestSuite) TestUpdateRole() {
    s.Run("Success", func() {
        roleID := uuid.New()
        newPermissions := []uuid.UUID{uuid.New(), uuid.New()}

        // Mock the role fetch
        s.mock.ExpectQuery(`SELECT .* FROM "roles"`).
            WithArgs(roleID).
            WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
                AddRow(roleID, "Old Name"))

        // Mock the transaction for update
        s.mock.ExpectBegin()

        // Mock the role update
        s.mock.ExpectExec(`UPDATE "roles"`).
            WithArgs("New Name", "New Description", sqlmock.AnyArg(), roleID).
            WillReturnResult(sqlmock.NewResult(1, 1))

        // Mock permission deletion and insertion
        s.mock.ExpectExec(`DELETE FROM "role_permissions"`).
            WithArgs(roleID).
            WillReturnResult(sqlmock.NewResult(0, 2))

        s.mock.ExpectExec(`INSERT INTO "role_permissions"`).
            WithArgs(sqlmock.AnyArg()).
            WillReturnResult(sqlmock.NewResult(2, 2))

        s.mock.ExpectCommit()

        // Execute the update
        role, err := s.repo.UpdateRole(context.Background(), roleID, "New Name", "New Description", newPermissions)

        s.NoError(err)
        s.Equal("New Name", role.Name)
    })
}
```

### AAA Comments

Use AAA comments for clarity:

```go
func (s *UserServiceTestSuite) TestDeleteUser() {
    s.Run("Success", func() {
        // Arrange
        userID := uuid.New()
        orgID := uuid.New()

        s.mockUserRepo.On("GetUserByID", mock.Anything, userID).
            Return(&schemas.User{ID: userID}, nil)
        s.mockUserRepo.On("DeleteUser", mock.Anything, userID).
            Return(nil)

        // Act
        err := s.service.DeleteUser(context.Background(), orgID, userID)

        // Assert
        s.NoError(err)
        s.mockUserRepo.AssertExpectations(s.T())
    })
}
```

## Build Tags

Go build tags control which test files are compiled for each test mode. Every test file must have the appropriate build tag.

### Unit Test Files

All unit test files must exclude integration builds:

```go
//go:build !integration

package mypackage
```

This ensures unit tests run with `go test ./...` (no tags) and `go test -tags=integration_all ./...`, but NOT with `go test -tags=integration ./...`.

### Integration Test Files

Integration tests require a real database and/or Redis. Use one of:

```go
//go:build integration || integration_all

package mypackage
```

This runs with both `go test -tags=integration` and `go test -tags=integration_all`.

For tests that should ONLY run in the full suite (e.g., cross-package tests):

```go
//go:build integration_all

package mypackage
```

### How Make Targets Map to Tags

| Command | Tags | What Runs |
|---------|------|-----------|
| `make test` | (none) | Unit tests only (`!integration` files) |
| `make test-integration` | `integration` | Integration tests only (`integration \|\| integration_all` files) |
| `make test-all` | `integration_all` | Both unit tests (`!integration`) AND integration tests (`integration \|\| integration_all`) |

### Legacy Format

Some older files may also include the `// +build` comment (pre-Go 1.17 syntax). Both formats are accepted, but prefer `//go:build` for new files:

```go
//go:build !integration
// +build !integration
```

---

## Running Tests

### Basic Test Commands

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./pkg/database/repositories

# Run specific test suite
go test ./pkg/database/repositories -run TestUserRepositoryTestSuite

# Run specific test
go test ./pkg/database/repositories -run TestUserRepositoryTestSuite/TestGetUserByID

# Run specific subtest
go test ./pkg/database/repositories -run TestUserRepositoryTestSuite/TestGetUserByID/Success
```

### Verbose Output

```bash
# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Database Mode

```bash
# Run with mock database (unit tests only)
go test ./...

# Run with test database (integration tests only)
go test -tags=integration ./...

# Run BOTH unit and integration tests
go test -tags=integration_all ./...

# Set test database environment variables for integration tests
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5432
export TEST_DB_USER=test_user
export TEST_DB_PASSWORD=test_password
export TEST_DB_NAME=industrix_test

# Example: Run integration tests with environment variables
TEST_DB_HOST=localhost TEST_DB_PORT=5432 TEST_DB_USER=test_user \
TEST_DB_PASSWORD=test_password TEST_DB_NAME=industrix_test \
go test -tags=integration ./...
```

### Build Tag Reference

| Test Type | Build Tag | Runs With No Tag | Runs With `-tags=integration` | Runs With `-tags=integration_all` |
|-----------|-----------|------------------|-------------------------------|-----------------------------------|
| Unit tests (mock) | `//go:build !integration` | ✅ Yes | ❌ No | ✅ Yes |
| Integration tests (real DB) | `//go:build integration \|\| integration_all` | ❌ No | ✅ Yes | ✅ Yes |

**Summary**:
- **No tags**: Unit tests only (fast, no database required)
- **`-tags=integration`**: Integration tests only (requires database)
- **`-tags=integration_all`**: BOTH unit and integration tests (comprehensive testing)

### Makefile Targets

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run all tests (mock + integration)
make test-all

# Run tests with race detector
make test-race
```

### CI/CD Integration

Tests run automatically in GitHub Actions on:
- Every push to any branch
- Every pull request
- Before merge to main

Coverage reports are generated and available in the Actions tab.

## Best Practices

### 1. Test Independence

Tests should not depend on each other:

```go
✅ GOOD: Each test is independent
func (s *UserRepositoryTestSuite) TestGetUserByID() {
    s.Run("Success", func() {
        userID := uuid.New()
        // Setup specific to this test
        s.mock.ExpectQuery("SELECT").WithArgs(userID).WillReturnRows(...)

        user, err := s.repo.GetUserByID(context.Background(), userID)

        s.NoError(err)
    })
}

❌ BAD: Tests depend on shared state
var sharedUserID uuid.UUID  // Avoid shared state between tests

func (s *UserRepositoryTestSuite) TestCreateUser() {
    user, _ := s.repo.CreateUser(...)
    sharedUserID = user.ID  // Setting shared state
}

func (s *UserRepositoryTestSuite) TestGetUserByID() {
    user, _ := s.repo.GetUserByID(context.Background(), sharedUserID)  // Depends on previous test
}
```

### 2. Test One Thing

Each test should focus on one behavior:

```go
✅ GOOD: Tests specific behavior
func (s *UserRepositoryTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        // Tests only successful creation
    })

    s.Run("DuplicateEmail", func() {
        // Tests only duplicate email error
    })
}

❌ BAD: Tests multiple behaviors
func (s *UserRepositoryTestSuite) TestUser() {
    // Creates user, updates user, deletes user all in one test
}
```

### 3. Clear Test Names

Use descriptive names that explain the scenario:

```go
✅ GOOD: Clear, descriptive names
s.Run("Success", func() { ... })
s.Run("NotFound", func() { ... })
s.Run("EmptyEmail", func() { ... })
s.Run("DuplicateKey", func() { ... })
s.Run("ContextTimeout", func() { ... })

❌ BAD: Vague, unclear names
s.Run("Test1", func() { ... })
s.Run("Error", func() { ... })
s.Run("Case2", func() { ... })
```

### 4. Test Coverage

Aim for high coverage but focus on meaningful tests:

```go
✅ GOOD: Test meaningful scenarios
- Happy path (success case)
- All error paths
- Edge cases (empty, nil, boundary values)
- Context timeout
- Concurrent operations (if applicable)

❌ BAD: Test for coverage numbers only
- Only testing getters/setters
- Only testing successful paths
- Ignoring error handling
```

### 5. Mock Verification

Always verify mock expectations:

```go
✅ GOOD: Verify all mocks called correctly
func (s *UserServiceTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        s.mockUserRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*schemas.User")).
            Return(nil)

        user, err := s.service.CreateUser(context.Background(), "test@example.com", "Test User")

        s.NoError(err)
        s.mockUserRepo.AssertExpectations(s.T())  // Verify mock was called
    })
}

❌ BAD: No verification
func (s *UserServiceTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        s.mockUserRepo.On("CreateUser", ...).Return(nil)

        user, err := s.service.CreateUser(...)

        s.NoError(err)
        // Missing: s.mockUserRepo.AssertExpectations(s.T())
    })
}
```

### 6. Cleanup and Isolation

Clean up resources and maintain test isolation:

```go
✅ GOOD: Proper cleanup
func (s *UserRepositoryTestSuite) SetupTest() {
    // Setup fresh state for each test
    s.db = setupTestDB()
}

func (s *UserRepositoryTestSuite) TearDownTest() {
    // Clean up after each test
    cleanupTestDB(s.db)
}

✅ GOOD: Use defer for cleanup in integration tests
func (s *UserRepositoryTestSuite) TestCreateUser() {
    s.Run("Success", func() {
        user := &schemas.User{Email: "test@example.com"}
        s.db.Create(user)
        defer s.db.Delete(user)  // Clean up test data

        result, err := s.repo.GetUserByID(context.Background(), user.ID)
        s.NoError(err)
    })
}
```

### 7. Realistic Test Data

Use realistic test data that represents actual use cases:

```go
✅ GOOD: Realistic test data
user := &schemas.User{
    Email: "john.doe@example.com",
    Name:  "John Doe",
}

❌ BAD: Unrealistic or confusing test data
user := &schemas.User{
    Email: "a",
    Name:  "x",
}
```

### 8. Avoid Testing Implementation Details

Test behavior, not implementation:

```go
✅ GOOD: Test public API behavior
func (s *UserServiceTestSuite) TestCreateUser() {
    user, err := s.service.CreateUser(context.Background(), "test@example.com", "Test User")

    s.NoError(err)
    s.NotNil(user)
    s.Equal("test@example.com", user.Email)
}

❌ BAD: Test internal implementation
func (s *UserServiceTestSuite) TestCreateUser() {
    // Don't test if service calls specific private methods
    // Don't test internal state that's not exposed
}
```

---

## Summary

This document established comprehensive unit testing conventions for the Industrix backend:

- **Framework**: testify/suite with organized test suites
- **Coverage**: 80% for repository and service layers
- **Mocking**: mockery for interface mocks, sqlmock for database
- **Database Testing**: Three modes (mock-only, test DB, both)
- **Patterns**: AAA structure, subtests, comprehensive error testing
- **Best Practices**: Independence, clarity, realistic data

For questions or suggestions, contact the development team.
