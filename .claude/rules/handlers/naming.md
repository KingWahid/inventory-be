---
paths:
  - "services/*/api/**"
  - "sync-services/*/server/**"
---

# Handler Layer Naming

## Handler Struct

Named `ServerHandler`. Implements the generated `stub.ServerInterface`:

```go
type ServerHandler struct {
    service siteservice.Service
}

func NewServerHandler(svc siteservice.Service) stub.ServerInterface {
    return &ServerHandler{service: svc}
}
```

Inject service interfaces — never concrete implementations.

## Handler Methods

Method names match the generated stub interface (from OpenAPI spec). The handler's job is:
1. Bind and validate request
2. Convert stub types to domain types
3. Call service
4. Convert domain types to stub types
5. Return JSON response

```go
func (h *ServerHandler) PostResource(ctx echo.Context) error {
    var req stub.CreateRequest
    if err := common.BindRequestBody(ctx, &req); err != nil {
        return err
    }
    domainReq := FromStubCreateRequest(&req)
    response, err := h.service.Create(ctxTimeout, *domainReq)
    if err != nil {
        return err
    }
    return ctx.JSON(http.StatusCreated, ToStubResponse(response))
}
```

## Converters (`converters.go`)

Organized in three sections with separator comments:

```go
// =============================================================================
// Request Converters (stub -> domain)
// =============================================================================

func FromStubCreateRequest(r *stub.CreateRequest) *types.CreateRequest { ... }

// =============================================================================
// Response Converters (domain -> stub)
// =============================================================================

func ToStubResponse(r *types.Response) *stub.Response { ... }

// =============================================================================
// Internal Helper Converters
// =============================================================================

func toStubItems(items []types.Item) []stub.Item { ... }  // unexported
```

**Naming**:
- Request converters: `FromStub{StubType}` — exported
- Response converters: `ToStub{DomainType}` — exported
- Internal helpers: `toStub{Type}`, `toStub{Type}Ptr` — unexported (lowercase)

**Rules**:
- All `FromStub*` functions take pointer parameters (`*stub.X`)
- All converters check nil and return nil early
- Return pointers (`*types.X`) for consistency
- Dereference at the call site when service expects value: `*domainReq`

## Stub Types (`stub/`)

Generated from OpenAPI spec. **Never edit** files in `stub/`. Regenerate with code generation tools in `tools/`.

## File Locations

```
services/{name}/api/
├── handlers.go       # HTTP handler methods
├── converters.go     # Stub <-> domain converters
└── init_test.go      # Test suite setup (integration)
```
