# Service Layer Unit Test Guide

## Overview

This document explains how unit tests are structured for the service layer. All tests are **unit tests only** - they mock all dependencies and test business logic in isolation.

## Test Structure

Following the convention from `docs/conventions/unit-test.md`, service layer tests use a **split pattern**:

- `service_test.go` - Test suite definition, setup/teardown, helper functions
- `*_test.go` - Individual test files for each service file (e.g., `devices_test.go` for `devices.go`)

## File Structure

Each service package should have:

1. **`service_test.go`** - Test suite with:
   - Test suite struct definition with repository mocks
   - Transaction manager mock (imported from `pkg/database/transaction/mocks`)
   - JWT encoder mock (if needed)
   - Setup/teardown methods
   - Helper functions for creating test data

2. **`<feature>_test.go`** - Test files for each feature file:
   - One test file per source file (e.g., `users_test.go` for `users.go`)
   - Contains test methods that use the shared `ServiceTestSuite`

## Test Pattern

Each test method follows the **AAA pattern** (Arrange-Act-Assert):

```go
func (s *ServiceTestSuite) TestMethodName() {
    s.Run("Success", func() {
        // Arrange - Setup mocks and test data
        s.mockRepo.On("Method", ...).Return(...).Once()
        
        // Act - Call service method
        result, err := s.service.Method(...)
        
        // Assert - Verify results
        s.NoError(err)
        s.NotNil(result)
    })
    
    s.Run("ErrorCase", func() {
        // Test error scenarios
    })
}
```

## Transaction Manager Mock

Service layer tests use a centralized transaction manager mock located at `pkg/database/transaction/mocks/Manager.go`. This mock:

1. Records mock expectations using `testify/mock`
2. Actually executes the transaction function `fn(ctx)` after expectations are checked
3. Provides cleanup via `NewManager(t)` which auto-asserts expectations

### Usage Pattern

```go
import (
    txMocks "github.com/industrix-id/backend/pkg/database/transaction/mocks"
)

type ServiceTestSuite struct {
    suite.Suite
    service       *service
    mockTxManager *txMocks.Manager
    // ... other mocks
}

func (s *ServiceTestSuite) SetupTest() {
    s.mockTxManager = txMocks.NewManager(s.T())

    // Default expectation: allow transactions and execute the function
    s.mockTxManager.On("RunInTx", mock.Anything, mock.Anything).Return(nil).Maybe()

    s.service = &service{
        txManager: s.mockTxManager,
        // ... other dependencies
    }
}
```

**Note**: Do NOT define a local `mockTransactionManager` in `service_test.go`. Always import from the centralized location.

## Adding Tests to a Service

### 1. Regenerate Mocks (if needed)

If repository interfaces have changed, regenerate mocks:

```bash
make mocks-generate
```

### 2. Create Test Files

For each service file, create a corresponding test file:

- `users.go` → `users_test.go`
- `create_user.go` → `create_user_test.go`
- `update_user.go` → `update_user_test.go`
- `delete_user.go` → `delete_user_test.go`

### 3. Test Coverage Requirements

Aim for **80% coverage** as per conventions. Test:

- ✅ Success paths
- ✅ Error paths (repository errors, validation errors, etc.)
- ✅ Edge cases (nil parameters, empty results, etc.)
- ✅ Auth token methods (JWT decode success/failure)
- ✅ Context cancellation/timeout

### 4. Running Tests

```bash
# Run all unit tests (excludes integration tests)
go test ./pkg/services/...

# Run tests for a specific service
go test ./pkg/services/identity/...

# Run specific test
go test ./pkg/services/identity -run TestServiceTestSuite/TestCreateUser

# Run with coverage
go test -cover ./pkg/services/...
```

## Example Test Scenarios

For each service method, test:

1. **Success** - Happy path with valid data
2. **Success_WithNilParameters** - When optional params are nil
3. **EmptyResult** - When repository returns empty results
4. **RepositoryError** - When repository returns error
5. **ContextCancelled** - When context is cancelled
6. **InvalidAuthToken** - For `*ByAuthToken` methods, test invalid JWT
7. **JWTDecodeSuccess_ButRepositoryError** - JWT valid but repo fails

## Notes

- All tests use `//go:build !integration` to exclude from integration test runs
- Mocks are automatically verified by mockery v2 via cleanup functions
- Use `s.Run()` for subtests to organize test cases
- Use helper functions from `service_test.go` to create test data
- See existing service tests (e.g., `pkg/services/identity/`, `pkg/services/platform/`) as templates

