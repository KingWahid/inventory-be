# ADR 001 — HTTP domain errors (`AppError`)

**Decision:** Extend the existing [`AppError`](../../pkg/common/errorcodes/errors.go) struct (fluent `With*` methods, optional `message_id`) instead of introducing a parallel `CustomError` type. All services map errors through [`ToHTTP`](../../pkg/common/errorcodes/http.go); JSON errors to clients use the ARCHITECTURE §9 envelope (`success`, `error`, `meta`) via [`WriteHTTPError`](../../pkg/common/errorcodes/envelope.go).

**Context:** [`docs/conventions/codebase-conventions.md`](../conventions/codebase-conventions.md) refers to `common.NewCustomError`; this codebase standardizes on `AppError` + [`Problem`](../../pkg/common/errorcodes/errors.go) entry points so there is a single classification path and no undocumented parallel system.

