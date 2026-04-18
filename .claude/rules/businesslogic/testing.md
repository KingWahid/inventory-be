---
paths:
  - "pkg/services/**/*_test.go"
  - "pkg/common/**/*_test.go"
  - "pkg/eventbus/**/*_test.go"
  - "workers/jobs/consumers/**/*_test.go"
---

# Business Logic Layer Testing

## Service Unit Tests (`*_test.go`)

Build tag: `//go:build !integration`

Use testify/suite. Mock all repository dependencies:

```go
type CreateOrganizationTestSuite struct {
    suite.Suite
    service  *service
    mockRepo *mocks.Repository
}

func (s *CreateOrganizationTestSuite) SetupTest() {
    s.mockRepo = mocks.NewRepository(s.T())
    s.service = &service{
        orgRepo: s.mockRepo,
    }
}

func TestCreateOrganizationTestSuite(t *testing.T) {
    suite.Run(t, new(CreateOrganizationTestSuite))
}
```

## Mocking Repositories

Use generated mocks from `{repo}/mocks/Repository.go`:

```go
s.mockRepo.On("GetOrganizationByID",
    mock.MatchedBy(func(ctx context.Context) bool { return true }),
    testOrgID,
).Return(&schemas.Organization{ID: testOrgID}, nil)
```

After test, verify calls:
```go
s.mockRepo.AssertCalled(s.T(), "GetOrganizationByID", mock.Anything, testOrgID)
```

## Test Structure

One test suite per operation file. Subtests for scenarios:

```go
func (s *Suite) TestCreateOrganization_Success() { ... }
func (s *Suite) TestCreateOrganization_ValidationError() { ... }
func (s *Suite) TestCreateOrganization_DuplicateEmail() { ... }
```

## Assertions

Verify CustomError properties when testing error paths:

```go
s.Error(err)
customErr, ok := err.(*common.CustomError)
s.True(ok)
s.Equal(http.StatusBadRequest, customErr.HTTPCode())
s.Equal(int(errorcodes.BadRequestForm), customErr.Code())
```

## Consumer Handler Tests (`workers/jobs/consumers/`)

Use table-driven tests with mock services:

```go
tests := []struct {
    name          string
    payload       *payloadStruct
    invalidJSON   bool
    serviceError  error
    expectedError bool
    errorContains string
}{
    {name: "success", ...},
    {name: "invalid JSON", invalidJSON: true, expectedError: true},
    {name: "invalid UUID", expectedError: true, errorContains: "invalid subscription_id"},
    {name: "service error", serviceError: someErr, expectedError: true},
}
```

Create mock services with function fields for flexible behavior:

```go
type mockFeatureService struct {
    feature.ConsumerService
    enableFunc func(ctx context.Context, orgID, subID, verID uuid.UUID) error
}
```
