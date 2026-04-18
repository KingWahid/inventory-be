# Git Commit Conventions

This document defines commit message conventions for the Industrix Backend project, based on analysis of existing commit history and Conventional Commits specification.

## Commit Message Format

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

## Commit Types

Required in all commits. Use these types based on the nature of the change:

### Primary Types

| Type | Purpose | When to Use |
|------|---------|-------------|
| `feat` | New features | Adding new functionality or capabilities |
| `fix` | Bug fixes | Fixing broken functionality or errors |
| `test` | Tests | Adding or modifying tests |
| `refactor` | Code restructuring | Changing code structure without altering behavior |
| `docs` | Documentation | Adding or updating documentation |
| `style` | Code style | Formatting, whitespace, comments (no logic change) |
| `chore` | Maintenance | Dependencies, configuration, build tasks |
| `perf` | Performance | Performance improvements or optimizations |
| `ci` | CI/CD | Changes to CI/CD pipelines or workflows |

### Less Common Types

| Type | Purpose | Example |
|------|---------|---------|
| `build` | Build system | Changes to build configuration |
| `revert` | Revert commits | Reverting previous commits |

## Scope Guidelines

Scope is **optional but recommended** for clarity. It indicates which part of the codebase is affected.

### Common Scopes (Analysis of 200 Recent Commits)

**Most Frequent Scopes**:
1. `ci` (29 occurrences) - CI/CD pipeline changes
2. `linter` (14 occurrences) - Linter configuration or fixes
3. `tests` (10 occurrences) - Test infrastructure or test files
4. `ftm` (9 occurrences) - Fuel Tank Monitoring module
5. `repository` (10 occurrences) - Repository layer
6. `test` (7 occurrences) - Individual test additions
7. `rbac` (6 occurrences) - Role-Based Access Control
8. `deps` (3 occurrences) - Dependency updates
9. `godot` (3 occurrences) - Godot linter fixes

### Scope Categories

**Architecture Layers**:
```
repository  - Data access layer
service     - Business logic layer
api         - API/controller layer
schemas     - Database schemas
```

**Feature Domains**:
```
ftm         - Fuel Tank Monitoring
rbac        - Role-Based Access Control
auth        - Authentication
notification - Notifications
operation   - Operations
device      - Device management
```

**Infrastructure**:
```
ci          - CI/CD pipelines
infra       - Infrastructure code
config      - Configuration
build       - Build system
```

**Quality & Tools**:
```
tests       - Test infrastructure
test        - Individual tests
linter      - Linter configuration
lint        - Linting fixes
godot       - Godot linter
gosec       - Security linter
gocyclo     - Complexity linter
```

**Other**:
```
deps        - Dependencies
docs        - Documentation (can also be type)
Makefile    - Makefile changes
```

### Scope Usage Patterns

**With Scope** (Preferred when changes are focused):
```bash
feat(repository): add bulk retrieval methods
fix(ci): add Redis environment variables
test(repository): add comprehensive unit tests
refactor(rbac): consolidate update methods
docs(testing): require Redis integration testing
style(godot): end comments with periods
chore(deps): update go.mod for Redis mock
```

**Without Scope** (Acceptable for broad changes):
```bash
test: improve repository test coverage to 90%+
chore: update go.mod and go.sum
style: align comment formatting
docs: enhance README with setup instructions
```

**Multiple Scopes** (Use slash or list in body):
```bash
refactor(rbac/service): consolidate methods
refactor(sync+infra): update error handling

# Or use body for details:
refactor: comprehensive linter fixes

Reduced issues from 133 to 6 (95% reduction).
Affected areas: repositories, services, API handlers.
```

## Subject Line Rules

1. **Imperative Mood**: Use "add" not "added" or "adds"
   - ✅ `feat(api): add user endpoint`
   - ❌ `feat(api): added user endpoint`
   - ❌ `feat(api): adds user endpoint`

2. **No Capitalization**: Don't capitalize first word after colon
   - ✅ `feat(repository): add bulk methods`
   - ❌ `feat(repository): Add bulk methods`

3. **No Period**: Don't end with period
   - ✅ `fix(tests): resolve data race`
   - ❌ `fix(tests): resolve data race.`

4. **Length**: Keep under 72 characters
   - Use body for additional context if needed

5. **Be Specific**: Clearly state what changed
   - ✅ `fix(tests): resolve data race in BaseRepository cache tests`
   - ❌ `fix: fix tests`

## Body Guidelines

Use body when:
- Change needs explanation of **why** (not what - code shows that)
- Change affects multiple areas
- Change has important implications
- Breaking changes need documentation

**Format**:
- Separate from subject with blank line
- Wrap at 72 characters
- Use bullet points for lists
- Focus on motivation and context

**Example**:
```
feat(repository): enhance soft delete functionality with user tracking

Add deleted_by field tracking to soft delete operations across repositories.
This enables audit trails showing which user performed deletion actions.

Affected repositories:
- UserRepository
- OrganizationRepository
- DeviceRepository
```

## Footer Guidelines

Use footer for:
- Breaking changes: `BREAKING CHANGE: description`
- Issue references: `Closes #123`, `Fixes #456`, `Relates to #789`

**Example**:
```
feat(api): update authentication endpoint structure

BREAKING CHANGE: Auth endpoint now requires additional header
- Old: POST /api/auth with body
- New: POST /api/v2/auth with X-API-Version header

Closes #234
```

## Pre-Commit Validation

All commits are validated with pre-commit hooks that run:

1. **Code Formatting** (`make format`)
   - `goimports` - Import organization
   - `gofmt` - Code formatting

2. **Linting** (`make lint-fix`)
   - Multiple linters: godot, gosec, gocyclo, errcheck, etc.
   - Auto-fixes applied when possible

3. **Quick Tests** (`make test-short`)
   - Fast unit tests
   - Ensures code compiles and basic tests pass

**Bypassing Pre-Commit** (not recommended):
```bash
git commit --no-verify
```

## Common Patterns from Project History

### Pattern 1: Test Coverage Improvements

**Without scope** (general improvement):
```bash
test: improve repository test coverage to 90%+
```

**With scope** (specific component):
```bash
test(repository): add comprehensive unit tests for untested functions
feat(test): add integration tests for UserRepository
```

### Pattern 2: Linter Fixes

**Specific linter**:
```bash
style(godot): end comments with periods in caches, repositories, and services
fix(gosec): resolve security warnings in authentication
```

**Multiple linters** (use body):
```bash
chore(ci): fix lint errors

Fixes across multiple linters:
- errorlint: wrap errors properly
- gocritic: simplify conditionals
- staticcheck: remove unused code
- revive: fix exported comments
- godot: add periods to comments
```

### Pattern 3: CI/CD Changes

```bash
feat(ci): add integration test support with coverage
fix(ci): add Redis environment variables for integration tests
chore(ci): update workflow to use latest actions
```

### Pattern 4: Dependency Updates

```bash
chore(deps): update fuel-tank-monitoring module for pq array support
chore: update go.mod and go.sum for Redis mock dependency
chore(deps): run go mod tidy on all modules
```

### Pattern 5: Refactoring

**Service layer**:
```bash
refactor(rbac): consolidate UpdateRoleWithUser into UpdateRole
refactor(services): extract common validation logic
```

**Repository layer**:
```bash
refactor(repository): streamline quota record creation logic
refactor(error handling): standardize error code references across repositories
```

### Pattern 6: Feature Additions

**New functionality**:
```bash
feat(repository): add bulk retrieval methods for device and organization users
feat(repository): enhance soft delete functionality with user tracking
```

**Feature with tests**:
```bash
# Option 1: Separate commits
feat(repository): add bulk retrieval methods
test(repository): add tests for bulk retrieval methods

# Option 2: Combined (when tightly coupled)
feat(repository): add bulk retrieval methods with tests
```

### Pattern 7: Documentation

```bash
docs(testing): require Redis integration testing for all repositories
docs(conventions): add comprehensive unit test conventions
docs: enhance README with setup instructions
```

## Commit Organization Strategies

### Strategy 1: Separate Concerns

Split commits when:
- Mixing feature + documentation
- Mixing refactoring + new functionality
- Mixing different bug fixes
- Mixing test + implementation (sometimes - use judgment)

**Example**:
```bash
# Commit 1: Implementation
feat(repository): add soft delete user tracking

# Commit 2: Tests
test(repository): add tests for soft delete user tracking

# Commit 3: Documentation
docs(repository): document soft delete user tracking behavior
```

### Strategy 2: Atomic Features

Keep together when:
- Implementation and tests are tightly coupled
- Multiple files implement single cohesive feature
- Changes don't make sense separately

**Example**:
```bash
# Single commit for cohesive feature
feat(rbac): add role permission validation

Includes:
- RoleService validation logic
- RoleRepository query methods
- Unit and integration tests
```

### Strategy 3: Fix + Cleanup

```bash
# Commit 1: Fix the bug
fix(tests): resolve data race in BaseRepository cache tests

# Commit 2: Related cleanup (optional separate commit)
refactor(tests): improve test isolation and cleanup
```

## Checklist Before Committing

- [ ] Changes are logically grouped (one concern per commit)
- [ ] Commit message follows `<type>(<scope>): <subject>` format
- [ ] Subject line uses imperative mood
- [ ] Subject line is under 72 characters
- [ ] Body explains "why" if needed
- [ ] Pre-commit hooks pass (or intentionally bypassed with reason)
- [ ] Breaking changes documented in footer
- [ ] Related issues referenced in footer

## Tools and Automation

### Git Commit Organizer Agent

For complex changes affecting multiple areas, use the `git-commit-organizer` agent to:
- Analyze all uncommitted changes
- Propose logical commit groups
- Generate appropriate commit messages
- Execute commits in correct order

**Usage**: Claude Code will automatically suggest using this agent when you have complex uncommitted changes.

### Pre-Commit Hook

Located at `.githooks/pre-commit`, automatically runs:
```bash
make pre-commit  # format + lint-fix + test-short
```

**Setup** (if not already configured):
```bash
git config core.hooksPath .githooks
```

## References

- **Conventional Commits**: https://www.conventionalcommits.org/
- **Project Git Hooks**: `.githooks/pre-commit`
- **Git Commit Organizer**: `.claude/agents/git-commit-organizer.md`
