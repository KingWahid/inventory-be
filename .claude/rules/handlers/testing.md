---
paths:
  - "services/*/api/**/*_test.go"
  - "sync-services/*/server/**/*_test.go"
---

# Handler Layer Testing

## Handler tests are integration tests

Build tag: `//go:build integration || integration_all`

Handler tests run against a real Echo server with real DB and Redis — they test the full HTTP pipeline.

## Test Suite Structure

Inherit from `testutils.ServiceIntegrationTestSuite`:

```go
type OperationIntegrationTestSuite struct {
    testutils.ServiceIntegrationTestSuite
    Service siteservice.Service
    Handler *ServerHandler
}

type SiteCategoriesIntegrationTestSuite struct {
    OperationIntegrationTestSuite
}

func TestSiteCategoriesIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(SiteCategoriesIntegrationTestSuite))
}
```

## Setup

Initialize real service, create handler, start Echo server:

```go
func (s *OperationIntegrationTestSuite) SetupSuite() {
    s.ServiceIntegrationTestSuite.SetupSuite()  // DB, Redis, JWT, TestUser, TestOrg

    s.Service, _ = siteservice.NewService(siteservice.Dependencies{
        DB:          s.DB,
        RedisClient: s.RedisClient,
    })
    s.Handler = NewServerHandler(s.Service).(*ServerHandler)
    s.StartServer()
}
```

Register routes with middleware:

```go
func (s *OperationIntegrationTestSuite) StartServer() {
    s.ServiceIntegrationTestSuite.StartServer(func(e *echo.Echo) {
        group := e.Group("/operation")
        group.Use(initialization.JWTAuthMiddleware([]string{}))
        stub.RegisterHandlers(group, s.Handler)
    })
}
```

## Making Requests

Use the base suite's `CreateRequest` helper:

```go
req, _ := s.CreateRequest(http.MethodGet, "/site-categories", nil)
req.Header.Set("Authorization", "Bearer "+s.AuthToken)

resp, _ := http.DefaultClient.Do(req)
defer resp.Body.Close()

s.Equal(http.StatusOK, resp.StatusCode)
```

## Test Scenarios

Cover per endpoint:
- `Success` — valid request with auth
- `401 Unauthorized` — missing or invalid auth header
- `404 NotFound` — nonexistent resource ID
- `400 BadRequest` — invalid input
- `500 InternalServerError` — swap service with mock that returns error, restart server

For error injection, stop server, swap service, restart:

```go
s.StopServer()
s.Service = &mockService{err: someError}
s.Handler = NewServerHandler(s.Service).(*ServerHandler)
s.StartServer()
defer func() { /* restore original */ }()
```
