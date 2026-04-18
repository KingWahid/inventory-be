---
name: create-pr
description: Create a GitHub Pull Request with auto-generated description based on the PR template. Analyzes all branch changes, categorizes by layer, and fills out every section. Usage: /create-pr [JIRA-TICKET...]
---

# Create Pull Request

Analyze all changes on the current branch compared to the base branch, generate a PR description from the project template, and create the PR using `gh`.

## Arguments

- **Positional (optional)**: Jira ticket IDs (e.g., `IDX-123 IDX-456`). If not provided, auto-detect or prompt.
- **`--base <branch>`**: Base branch to compare against. Defaults to `main`.
- **`--draft`**: Create as draft PR.
- **`--title <title>`**: Override auto-generated title.

## Procedure

### Step 1: Validate Prerequisites

```bash
# Must be on a feature branch
CURRENT_BRANCH=$(git branch --show-current)
# Fail if on main or master

# Must have commits ahead of base
git rev-list --count main..HEAD
# Fail if 0

# Ensure branch is pushed (or push it)
git push -u origin "$CURRENT_BRANCH"

# Ensure gh is authenticated
gh auth status
```

### Step 2: Gather Branch Information

Run these commands to collect change data:

```bash
BASE_BRANCH="${BASE:-main}"

# All commits on this branch (for title generation and type detection)
git log ${BASE_BRANCH}...HEAD --oneline

# Full commit messages (for Jira extraction and detailed analysis)
git log ${BASE_BRANCH}...HEAD --format="%h %s%n%b---"

# Files changed with add/modify/delete status
git diff ${BASE_BRANCH}...HEAD --name-status

# Files changed with line counts
git diff ${BASE_BRANCH}...HEAD --stat
```

### Step 3: Categorize Changes by Layer

Map every changed file to a project layer using these path rules:

| Path Pattern | Layer | PR Template Section |
|---|---|---|
| `infra/database/migrations/` | Database migrations | Database / Data |
| `infra/database/cmd/seed/` | Seed / mock data | Database / Data |
| `pkg/database/schemas/` | Database schemas | Database / Data |
| `pkg/database/repositories/` | Repository layer | Service / Business Logic |
| `pkg/database/` (other) | Database utilities | Service / Business Logic |
| `pkg/services/` | Business logic | Service / Business Logic |
| `pkg/common/` | Shared utilities | Service / Business Logic |
| `pkg/eventbus/` | Event bus | Service / Business Logic |
| `services/*/openapi/` | OpenAPI specs | API / Contracts |
| `services/*/stub/` | Generated stubs | API / Contracts |
| `services/*/api/` | HTTP handlers | API / Contracts |
| `services/` (other: cmd, config, fx) | Service wiring | API / Contracts |
| `workers/` | Worker infrastructure | Service / Business Logic |
| `sync-services/` | MQTT sync services | Service / Business Logic |
| `infra/kong/` | Kong gateway | Configuration / infra |
| `infra/` (other, NOT migrations/seed) | Infrastructure | Configuration / infra |
| `docker-compose*.yaml` | Docker compose | Configuration / infra |
| `.github/workflows/` | CI/CD | Configuration / infra |
| `Makefile` | Build tooling | Configuration / infra |
| `.env*` | Environment config | Deployment / Rollback |
| `docs/` | Documentation | (mention in Scope only) |
| `.claude/` | Claude config | (mention in Scope only) |

**Test files** (`*_test.go`) belong to their parent layer but also inform the "How It Was Tested" section.

Build a **Scope** summary from the layers touched (e.g., "repositories, services, handlers, migrations, mock data").

### Step 4: Determine Type of Change

Apply these detection rules. Multiple types can be true simultaneously.

**From commit message prefixes:**

| Commit Prefix | Type of Change |
|---|---|
| `feat` | Feature |
| `fix` | Bug fix |
| `refactor` | Refactor / cleanup |
| `chore`, `ci` | Configuration / infra |
| `perf` | Feature (performance improvement) |
| `docs`, `style` | Refactor / cleanup |

**From file paths (supplements commit detection):**

| File Path Signal | Type of Change |
|---|---|
| New files in `infra/database/migrations/` | Database / migrations |
| New files in `services/*/api/handler_*` or `pkg/services/` | Feature |
| Changes in `infra/`, `docker-compose*`, `.github/`, `Makefile` | Configuration / infra |

**Breaking change detection:**
```bash
# Check commit messages for BREAKING CHANGE footer
git log ${BASE_BRANCH}...HEAD --format="%B" | grep -i "breaking change"

# Check for removed API paths in OpenAPI
git diff ${BASE_BRANCH}...HEAD -- 'services/*/openapi/openapi.yaml' | grep '^-.*/'
```

### Step 5: Collect Jira Tickets

Try these sources in order. Stop at the first match:

1. **Skill arguments**: If the user passed ticket IDs (e.g., `/create-pr IDX-123 IDX-456`)
2. **Branch name**: Extract pattern `[A-Z]+-\d+` from the current branch name
3. **Commit messages**: Extract pattern `[A-Z]+-\d+` from all commit subjects

If none found, ask the user:
> "No Jira tickets detected. Please provide Jira ticket ID(s) (e.g., IDX-123), or type 'none' to skip."

Format each ticket as:
```
- [IDX-123: <description>](https://industrix-team.atlassian.net/browse/IDX-123)
```

For the description, use the commit subject that references the ticket. If unavailable, use just the ticket ID as a placeholder and note it for the user to fill in.

### Step 6: Generate PR Title

If `--title` was not provided:

1. Identify the dominant commit type (most frequent prefix)
2. Identify the most common scope from commits
3. Summarize the change in imperative mood

Format: `<type>(<scope>): <summary>`

If Jira tickets exist, prepend up to 3 keys:
```
IDX-123: feat(auth): add organization-scoped JWT tokens
IDX-123 IDX-456: fix(billing): resolve duplicate payment processing
```

**Keep the total title under 72 characters.** If Jira prefix + title exceeds this, shorten the summary.

### Step 7: Read Targeted Diffs

Only read diffs for layers that have changes. Use path-scoped `git diff` to stay within context limits:

```bash
# Service / Business Logic notes
git diff ${BASE_BRANCH}...HEAD -- 'pkg/services/' 'pkg/common/' 'pkg/eventbus/'

# API / Contracts notes
git diff ${BASE_BRANCH}...HEAD -- 'services/*/api/' 'services/*/openapi/'

# Database / Data notes
git diff ${BASE_BRANCH}...HEAD -- 'infra/database/migrations/' 'pkg/database/schemas/'

# Environment variable changes
git diff ${BASE_BRANCH}...HEAD -- '.env*'

# Infrastructure changes
git diff ${BASE_BRANCH}...HEAD -- 'docker-compose*.yaml' 'infra/kong/' '.github/workflows/'
```

Skip sections where no files in that layer changed. For large diffs (50+ files in a layer), use `--stat` first and only read the most important files.

### Step 8: Generate PR Body

Fill each section of the PR template. The output must match the template's exact structure.

**IMPORTANT:** Do NOT include the `> [!WARNING]` banner from the template. It is a template hint, not PR content.

---

#### Section: PR Summary

```markdown
## PR Summary
- **Jira Ticket(s)**:
  - [IDX-123: ticket description](https://industrix-team.atlassian.net/browse/IDX-123)
- **What / Why**:
  <2-4 sentences synthesized from commit messages and diffs. Focus on WHAT changed and WHY.>
- **Scope**:
  <comma-separated list of layers touched, e.g., "repositories, services, handlers, migrations, mock data">
```

---

#### Section: Type of Change

Mark `[x]` for detected types from Step 4. Leave others as `[ ]`.

```markdown
## Type of Change

- [x] Feature
- [ ] Bug fix
- [ ] Refactor / cleanup
- [x] Database / migrations
- [ ] Configuration / infra
- [ ] Breaking change
```

---

#### Section: How It Was Tested

Analyze test file changes:
```bash
# Unit tests changed/added
git diff ${BASE_BRANCH}...HEAD --name-only | grep '_test.go$' | grep -v '_integration_test.go$'

# Integration tests changed/added
git diff ${BASE_BRANCH}...HEAD --name-only | grep '_integration_test.go$'

# Locale files changed
git diff ${BASE_BRANCH}...HEAD --name-only | grep 'locales/'
```

**Checkbox rules:**
- **Manual checkboxes**: ALWAYS leave unchecked. These are the human reviewer's responsibility.
- **Unit tests**: `[x]` only if unit test files were changed or added
- **Integration / e2e / service tests**: `[x]` only if integration test files were changed or added
- **Not applicable**: `[x]` only if zero test files changed AND the change is config/docs/infra only. Include explanation.
- **Locale tested**: `[x]` only if files in `pkg/common/translations/locales/` were changed

**Notes**: List which test files were added or modified, grouped by package.

---

#### Section: Service / Business Logic

Check boxes based on what is detected in the diffs:

| Checkbox | Detection Rule |
|---|---|
| New or changed service methods | Files in `pkg/services/` were modified |
| Transactions used appropriately | Diffs contain `transaction.WithTx`, `transaction.Manager`, or `transaction.GetDB` |
| Error handling uses common error codes | Diffs contain `common.NewCustomError`, `errorcodes.`, or `WithMessageID` |
| Cross-service impacts considered | Changes span 2+ service packages OR touch `pkg/eventbus/` |

If no files in the business logic layer changed, leave all unchecked and write "N/A" in Notes.

**Notes**: Summarize from the targeted diffs:
- New or changed service methods and their purpose
- Key business rules or validation logic
- Event-driven flows (publish/consume)
- Permission or RBAC implications

---

#### Section: API / Contracts

| Checkbox | Detection Rule |
|---|---|
| OpenAPI updated | Any `openapi.yaml` file changed |
| Request/response payloads backward compatible | `[x]` unless breaking changes detected in Step 4 |
| Error codes and translations aligned | Diffs contain `WithMessageID` AND locale files changed |

If no API layer files changed, leave all unchecked and write "N/A" in Notes.

**Notes**: Summarize:
- New endpoints added (path + method)
- Changed request/response shapes
- New error codes

---

#### Section: Database / Data

| Checkbox | Detection Rule |
|---|---|
| New migrations added and ordered correctly | New `.up.sql` / `.down.sql` files in `infra/database/migrations/` |
| Migrations tested up/down locally | ALWAYS leave unchecked (reviewer must verify) |
| Seeders / mock data adjusted | Files in `infra/database/cmd/seed/mock-data/` changed |
| Data backfill / cleanup plan documented | Diff contains UPDATE/backfill logic in migrations |

If no database layer files changed, leave all unchecked.

For new migrations, summarize the SQL: table names, columns, constraints, indexes.

---

#### Section: Deployment / Rollback

- **Target env**: `dev` (default). Use `staging` or `prod` if branch name contains `release`, `hotfix`, or `prod`.
- **Config / env vars**: List new or changed variables from `.env*` diffs. Write "none" if no changes.
- **Data / migrations**: Summarize (e.g., "3 new migrations: create_xxx table, add_yyy column, seed_zzz data"). Write "none" if no migrations.
- **Rollback**: Auto-generate:
  - If migrations: `Rollback migrations: make -C infra/database down n=<count>`
  - If code only: `Revert merge commit`
  - If config only: `Revert config changes and redeploy`
  - If mixed: Combine relevant steps

---

#### Section: Quick Checklist

**ALWAYS leave all checkboxes unchecked.** These are for the human reviewer to verify after reviewing the code.

```markdown
## Quick Checklist

- [ ] Code formatted (gofmt / goimports, linters clean) & no stray debug logs
- [ ] Logs and errors use common helpers & error codes where relevant
- [ ] Docs / OpenAPI / Postman collections updated if relevant
- [ ] Reviewers + labels set
```

### Step 9: Create the PR

```bash
gh pr create \
  --title "<title from Step 6>" \
  --body "$(cat <<'EOF'
<full PR body from Step 8>
EOF
)" \
  --base "${BASE_BRANCH}" \
  ${DRAFT:+--draft}
```

**IMPORTANT:** Always use HEREDOC (`cat <<'EOF'`) for the body. The PR body contains markdown with brackets, pipes, and backticks that break shell quoting.

### Step 10: Report Result

Output:
- The PR URL
- Title used
- Types of change detected
- Layers affected
- Number of files changed

Remind the user:
> "Please review the PR description. Manual testing checkboxes and Quick Checklist items are intentionally unchecked for you to verify."

## Important Rules

1. **Never fabricate test results.** Only check automated test boxes if test files were actually changed. Never check manual testing boxes.
2. **Never check Quick Checklist items.** Those are for human reviewers.
3. **Read diffs selectively.** Use `git diff -- <path>` per layer. Never dump the entire diff.
4. **Match the template exactly.** Section headers, checkbox format, and structure must be identical to `.github/pull_request_template.md`.
5. **Ask for Jira tickets if not found.** Do not silently skip Jira references.
6. **PR title under 72 characters.**
7. **Use HEREDOC for body.** Always `cat <<'EOF'` to avoid shell escaping issues.
8. **Exclude the WARNING banner.** The `> [!WARNING]` block in the template is a hint, not content.

## Service Name Reference

For Scope descriptions:

| Path | Service Name |
|---|---|
| `services/authentication/` | Authentication |
| `services/billing/` | Billing |
| `services/common/` | Common |
| `services/ftm/` | Fuel Tank Monitoring |
| `services/notification/` | Notification |
| `services/operation/` | Operation |
| `sync-services/ftm/` | FTM Sync (MQTT) |
| `workers/` | Background Workers |

## Jira URL Pattern

```
https://industrix-team.atlassian.net/browse/{TICKET_ID}
```

Supported project key pattern: uppercase letters followed by dash and digits (e.g., `IDX-123`, `PLAT-456`).

## Additional Resources

See [EXAMPLES.md](EXAMPLES.md) for:
- Real PR creation examples by scenario
- Generated PR body examples
- Edge case handling
