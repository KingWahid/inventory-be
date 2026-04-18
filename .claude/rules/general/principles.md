# General Principles

These principles govern all code in this codebase. Apply them in every change.

## DRY (Don't Repeat Yourself)

Eliminate duplication of logic. If the same behavior exists in multiple places, extract it into a shared abstraction. When a pattern is used across 40+ repositories (e.g., caching, transaction detection), it belongs in a base module — not copy-pasted.

## Single Source of Truth

Every piece of knowledge must have one authoritative location. Database schemas define data structure. OpenAPI specs define API contracts. Validation rules live at the boundary that owns them — downstream layers trust the contract, they don't re-validate.

## Separation of Concerns

Each layer handles exactly one responsibility. Data access does not know about HTTP. Business logic does not know about API stubs. Transport handles conversion at the boundary. If a function mixes concerns, split it.

## Single Responsibility

Each function, method, and module does one thing. A method that fetches data should not also validate authorization. A method that checks a flag should not also fetch related entities. Granular methods are testable, composable, and replaceable.

## Fail Fast

Validate preconditions at the earliest possible point. Services validate configuration at initialization — not at first use. Repositories check business rules (e.g., "must be deactivated before deletion") before attempting mutations. Return errors immediately rather than allowing invalid state to propagate.

## Explicit Over Implicit

Dependencies, state, and side effects must be visible. Context is always passed as the first parameter — never stored globally. Transactions propagate through context explicitly. Error codes are set through explicit method chaining. No hidden behavior.

## Defensive Programming

Assume inputs can be nil, duplicated, or malformed. Check nil before dereferencing. Normalize data (e.g., lowercase emails) before cache key generation. Use atomic operations for idempotency. Handle failed events through dead letter queues rather than silently dropping them.

## Composition Over Inheritance

Build complex behavior by composing small, focused components. Repositories embed a base module for shared behavior. Services depend on repository interfaces — not concrete types. FX modules compose the application graph from independent pieces.

## Dependency Inversion

Depend on interfaces, not implementations. All repository and service contracts are defined as interfaces. Concrete implementations are private. The dependency injection framework (FX) resolves the wiring — callers never instantiate dependencies directly.

## Least Privilege

Default to the most restrictive scope. All data access is organization-scoped by default. Unscoped access is the explicit exception, not the rule. Methods that bypass scoping are clearly named and documented as admin-only.

## Read Before Writing

Before modifying any directory, read its `README.md` first (if one exists). READMEs contain context about the module's purpose, constraints, and conventions that may not be obvious from the code alone. Skipping this leads to changes that violate local assumptions.

## Flag Rule Violations and New Patterns

When you encounter existing code that doesn't align with these rules, or when a task introduces something new (a new pattern, dependency, convention, or approach not covered by existing rules), you **must** raise it — never silently proceed.

- **Main agent / solo session**: Bring it up to the user directly before continuing.
- **Team member**: Report it to the team lead via message. Do not decide on your own whether the violation is acceptable.
- **Subagent**: Include it prominently in the result returned to the caller. Do not bury it.

This applies to both violations found in existing code and new patterns being introduced. The goal is to keep the codebase consistent — no silent drift.
