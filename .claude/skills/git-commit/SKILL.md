---
name: git-commit
description: Create well-structured Git commits following project conventions. Use when committing code changes with proper conventional commit messages.
---

# Git Commit Practices

## Quick Reference

**Format**: `<type>(<scope>): <description>`

**Common Types**:
- `feat` - New features or functionality
- `fix` - Bug fixes
- `test` - Test additions or modifications
- `refactor` - Code restructuring without behavior change
- `docs` - Documentation changes
- `style` - Code formatting, comments
- `chore` - Maintenance tasks (deps, config)
- `perf` - Performance improvements
- `ci` - CI/CD pipeline changes

## Creating Commits

### Single Logical Change

```bash
# 1. Check what's changed
git status
git diff

# 2. Stage relevant files
git add path/to/file1.go path/to/file2_test.go

# 3. Commit with conventional message
git commit -m "feat(repository): add bulk retrieval methods for device users"
```

### Multiple Logical Changes

Use the git-commit-organizer agent to group changes into logical commits:

```bash
# Claude Code will analyze changes and propose commit groups
# Then execute commits in logical order
```

## Pre-Commit Validation

The project uses pre-commit hooks that run:
- Code formatting (`make format`)
- Linting with auto-fix (`make lint-fix`)
- Quick tests (`make test-short`)

**Note**: These run automatically before each commit. To bypass (not recommended): `git commit --no-verify`

## Message Guidelines

**Structure**:
```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

**Subject Line**:
- Use imperative mood ("add" not "added" or "adds")
- Don't capitalize first letter after colon
- No period at end
- Max 72 characters

**Scope** (optional but recommended):
- Component/module affected: `repository`, `service`, `api`, `tests`, `ci`, `deps`
- Use singular form: `test` not `tests` (except for scope name itself)

**Body** (optional):
- Explain *why* not *what*
- Wrap at 72 characters

**Footer** (optional):
- Breaking changes: `BREAKING CHANGE: description`
- Issue references: `Closes #123`, `Fixes #456`

## Commit Organization

**Separate commits for**:
- Different features
- Bug fixes vs. features
- Refactoring vs. new functionality
- Documentation vs. code
- Test files (can be separate or with implementation)

**Combine in same commit**:
- Tightly coupled implementation files
- Tests with related code (when it makes sense)
- Multiple files implementing single feature

## Additional Resources

See [EXAMPLES.md](EXAMPLES.md) for:
- Real commit message examples from this project
- Common patterns by type (feat, fix, test, refactor, etc.)
- Multi-file commit examples
- Body and footer examples

See [CONVENTION.md](CONVENTION.md) for:
- Project-specific commit conventions
- Scope naming patterns
- Common commit patterns analysis
