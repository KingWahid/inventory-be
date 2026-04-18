# Git Commit Examples

Real commit message examples from the Industrix Backend project demonstrating conventional commit practices.

## Table of Contents

1. [Feature Commits (feat)](#feature-commits-feat)
2. [Bug Fix Commits (fix)](#bug-fix-commits-fix)
3. [Test Commits (test)](#test-commits-test)
4. [Refactoring Commits (refactor)](#refactoring-commits-refactor)
5. [Documentation Commits (docs)](#documentation-commits-docs)
6. [Style Commits (style)](#style-commits-style)
7. [Chore Commits (chore)](#chore-commits-chore)
8. [CI/CD Commits (ci)](#cicd-commits-ci)
9. [Performance Commits (perf)](#performance-commits-perf)
10. [Multi-File Commit Example](#multi-file-commit-example)

---

## Feature Commits (feat)

### Example 1: Repository Enhancement

```bash
git commit -m "feat(repository): add bulk retrieval methods for device and organization users"
```

**Files changed**: `pkg/repositories/device_user_repository.go`, `pkg/repositories/organization_user_repository.go`

**When to use**: Adding new functionality or capabilities to existing modules.

---

### Example 2: Test Infrastructure

```bash
git commit -m "feat(test): add comprehensive unit tests for UserRepository"
```

**Files changed**: `pkg/repositories/user_repository_test.go`

**When to use**: Adding new test coverage or test infrastructure.

---

### Example 3: CI/CD Feature

```bash
git commit -m "feat(ci): add integration test support with coverage for CI pipeline"
```

**Files changed**: `.github/workflows/test.yml`, `Makefile`

**When to use**: Adding new CI/CD capabilities or workflows.

---

### Example 4: Feature with Body

```bash
git commit -m "feat(repository): enhance soft delete functionality with user tracking

Add deleted_by field tracking to soft delete operations across repositories.
This enables audit trails showing which user performed deletion actions.

Affected repositories:
- UserRepository
- OrganizationRepository
- DeviceRepository"
```

**When to use**: Complex features requiring explanation of why and what changed.

---

## Bug Fix Commits (fix)

### Example 1: Test Fix

```bash
git commit -m "fix(tests): resolve data race in BaseRepository cache tests"
```

**Files changed**: `pkg/repositories/base_repository_test.go`

**When to use**: Fixing broken tests or test issues.

---

### Example 2: CI Configuration Fix

```bash
git commit -m "fix(ci): add Redis environment variables for integration tests"
```

**Files changed**: `.github/workflows/test.yml`

**When to use**: Fixing CI/CD pipeline issues.

---

### Example 3: Repository Logic Fix

```bash
git commit -m "fix(repository): remove obsolete deleted_at check from role_permissions query"
```

**Files changed**: `pkg/repositories/role_permission_repository.go`

**When to use**: Fixing incorrect logic or behavior in code.

---

### Example 4: RBAC Bug Fix

```bash
git commit -m "fix(rbac): resolve role update duplicate key error and translation issues"
```

**Files changed**: `pkg/services/rbac_service.go`, `pkg/repositories/role_repository.go`

**When to use**: Fixing bugs in specific feature domains.

---

## Test Commits (test)

### Example 1: Coverage Improvement

```bash
git commit -m "test: improve repository test coverage to 90%+"
```

**Files changed**: Multiple test files across repositories

**When to use**: General test coverage improvements without scope.

---

### Example 2: Specific Component Tests

```bash
git commit -m "test(repository): add comprehensive unit tests for untested functions"
```

**Files changed**: Various `*_repository_test.go` files

**When to use**: Adding tests for specific components or modules.

---

### Example 3: Integration Tests

```bash
git commit -m "test(repository): add comprehensive tests for FTM repositories"
```

**Files changed**: `pkg/repositories/ftm_*_repository_test.go`

**When to use**: Adding integration or E2E tests for modules.

---

## Refactoring Commits (refactor)

### Example 1: Test Code Refactoring

```bash
git commit -m "refactor(tests): improve test quality and code consistency"
```

**Files changed**: Multiple test files

**When to use**: Restructuring test code without changing behavior.

---

### Example 2: Service Consolidation

```bash
git commit -m "refactor(rbac): consolidate UpdateRoleWithUser into UpdateRole"
```

**Files changed**: `pkg/services/rbac_service.go`

**When to use**: Simplifying or consolidating code without changing functionality.

---

### Example 3: Error Handling Standardization

```bash
git commit -m "refactor(error handling): standardize error code references across repositories"
```

**Files changed**: Multiple repository files

**When to use**: Standardizing patterns across the codebase.

---

### Example 4: Repository Logic Streamline

```bash
git commit -m "refactor(repository): streamline quota record creation logic"
```

**Files changed**: `pkg/repositories/quota_repository.go`

**When to use**: Simplifying complex logic without changing behavior.

---

## Documentation Commits (docs)

### Example 1: Testing Documentation

```bash
git commit -m "docs(testing): require Redis integration testing for all repositories"
```

**Files changed**: `docs/testing/integration-tests.md`

**When to use**: Adding or updating documentation.

---

### Example 2: Build Tag Documentation

```bash
git commit -m "docs(tests): update build tag documentation to reflect integration_all behavior"
```

**Files changed**: `docs/testing/build-tags.md`

**When to use**: Clarifying or updating technical documentation.

---

### Example 3: Comprehensive Documentation

```bash
git commit -m "docs(conventions): add comprehensive unit test conventions and reorganize documentation"
```

**Files changed**: `docs/conventions/unit-testing.md`, reorganized docs structure

**When to use**: Major documentation additions or reorganization.

---

## Style Commits (style)

### Example 1: Comment Formatting

```bash
git commit -m "style: align comment in action_audit_log_test.go"
```

**Files changed**: `pkg/repositories/action_audit_log_test.go`

**When to use**: Formatting, whitespace, or comment alignment changes.

---

### Example 2: Comment Standardization

```bash
git commit -m "style(godot): standardize comment formatting in cache key builders and repositories"
```

**Files changed**: Multiple files with comment formatting

**When to use**: Linter-driven style fixes (godot, gofmt, etc.).

---

### Example 3: Bulk Comment Fixes

```bash
git commit -m "style(godot): end comments with periods in caches, repositories, and services"
```

**Files changed**: Multiple cache, repository, and service files

**When to use**: Applying style rules across multiple files.

---

## Chore Commits (chore)

### Example 1: Dependency Update

```bash
git commit -m "chore: update go.mod and go.sum for Redis mock dependency"
```

**Files changed**: `go.mod`, `go.sum`

**When to use**: Updating dependencies or package files.

---

### Example 2: Module Dependency Update

```bash
git commit -m "chore(deps): update fuel-tank-monitoring module for pq array support"
```

**Files changed**: `go.mod`, module dependency files

**When to use**: Updating specific module dependencies.

---

### Example 3: Cleanup Tasks

```bash
git commit -m "chore: format test files and update .gitignore"
```

**Files changed**: Test files, `.gitignore`

**When to use**: General maintenance, cleanup, or configuration updates.

---

### Example 4: Module Sync

```bash
git commit -m "chore(deps): run go mod tidy on all modules after test dependency update"
```

**Files changed**: Multiple `go.mod` and `go.sum` files

**When to use**: Synchronizing dependencies across modules.

---

## CI/CD Commits (ci)

### Example 1: Lint Error Fixes

```bash
git commit -m "chore(ci): fix lint errors (errorlint, gocritic, staticcheck, revive, godot); param rename in FTM API; prealloc; adjust cache comments; use embedded methods"
```

**Files changed**: Multiple files across codebase

**When to use**: Fixing CI linting issues across multiple linters.

**Note**: This is a complex commit - consider splitting if changes are in unrelated areas.

---

### Example 2: Test Cache Fix

```bash
git commit -m "fix(test): clean test cache in CI to prevent build tag contamination"
```

**Files changed**: `.github/workflows/test.yml`

**When to use**: Fixing CI/CD test execution issues.

---

### Example 3: Makefile Update

```bash
git commit -m "fix(Makefile): update goimports command to use the latest version via go run for improved compatibility"
```

**Files changed**: `Makefile`

**When to use**: Fixing build tool configurations.

---

## Performance Commits (perf)

### Example 1: Cache Optimization

```bash
git commit -m "perf(repository): improve cache key handling in FTM process summary methods"
```

**Files changed**: `pkg/repositories/ftm_process_summary_repository.go`

**When to use**: Performance improvements or optimizations.

---

## Multi-File Commit Example

### Scenario: Adding Integration Tests for a Repository

**Files to commit**:
- `pkg/repositories/user_repository_test.go` (new integration tests)
- `pkg/database/test_helpers.go` (test utilities)
- `docs/testing/integration-tests.md` (documentation)

**Recommended approach**: Split into 2 commits

**Commit 1 - Test Implementation**:
```bash
git add pkg/repositories/user_repository_test.go pkg/database/test_helpers.go
git commit -m "feat(test): add integration tests for UserRepository

Add comprehensive integration tests covering:
- CRUD operations with real database
- Foreign key constraint validation
- Soft delete functionality
- Cache invalidation behavior

Includes shared test helpers for database setup and teardown."
```

**Commit 2 - Documentation**:
```bash
git add docs/testing/integration-tests.md
git commit -m "docs(testing): add UserRepository integration test documentation"
```

**Rationale**:
- Separates code from documentation
- Test implementation is self-contained
- Documentation can be reviewed independently

---

## Commit Message Anti-Patterns

### ❌ Too Vague
```bash
git commit -m "fix stuff"
git commit -m "update tests"
git commit -m "changes"
```

### ❌ Wrong Tense
```bash
git commit -m "feat(api): added new endpoint"  # Should be "add"
git commit -m "fix(cache): fixing cache issue"  # Should be "fix"
```

### ❌ Missing Type
```bash
git commit -m "add user repository tests"  # Should be "test(repository): add user repository tests"
```

### ❌ Too Long Subject
```bash
git commit -m "feat(repository): add comprehensive integration tests with database setup, teardown, and validation for all CRUD operations"
# Should split into subject + body
```

### ✅ Correct Version
```bash
git commit -m "feat(repository): add comprehensive integration tests for UserRepository

Add integration tests covering:
- CRUD operations with real database
- Foreign key constraint validation
- Soft delete functionality
- Cache invalidation behavior"
```

---

## Tips for Good Commits

1. **Keep commits atomic**: One logical change per commit
2. **Write for the reviewer**: Clear subject explains the change
3. **Use body for context**: Explain why, not what (code shows what)
4. **Test before committing**: Pre-commit hooks will catch issues
5. **Use consistent scopes**: Follow project patterns (repository, service, api, tests, ci, deps)
6. **Capitalize scope**: Use lowercase after colon in subject
