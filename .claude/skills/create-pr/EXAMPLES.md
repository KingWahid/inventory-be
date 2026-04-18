# Create PR Examples

## Example 1: Feature with Jira Tickets Provided

**Invocation**: `/create-pr IDX-456 IDX-457`

**Branch**: `feat/organization-scoped-jwt`

**Commits**:
```
a1b2c3d feat(jwt): add organization name and logo to auth token claims
d4e5f6g feat(seed): add inline subquery support for mock data where clauses
h7i8j9k fix(seed): add platform_id filter to feature queries in mock data
l0m1n2o fix(migration): clean up subscription plan version features in down migration
```

**Generated title**: `IDX-456 IDX-457: feat(jwt): add org name and logo to auth tokens`

**Generated body** (key sections):

```markdown
## PR Summary
- **Jira Ticket(s)**:
  - [IDX-456: Add org context to JWT](https://industrix-team.atlassian.net/browse/IDX-456)
  - [IDX-457: Fix feature seed ambiguity](https://industrix-team.atlassian.net/browse/IDX-457)
- **What / Why**:
  Include organization name and logo in the authentication JWT so the frontend can display org context without extra API calls. Also fix mock data seed queries that were ambiguous when features exist on multiple platforms.
- **Scope**:
  JWT, authentication service, repositories, migrations, mock data, seed engine

---

## Type of Change

- [x] Feature
- [x] Bug fix
- [ ] Refactor / cleanup
- [ ] Configuration / infra
- [x] Database / migrations
- [ ] Breaking change

---

## How It Was Tested

- **Manual**:
  - [ ] Happy path
  - [ ] Edge cases
  - [ ] Regression on impacted flows
- **Automated**:
  - [x] Unit tests
  - [x] Integration / e2e / service tests
  - [ ] Not applicable
- **Others**:
  - [ ] Locale tested

Notes:
- Modified: `pkg/services/authentication/sign_in_test.go`, `choose_organization_test.go`
- Modified: `pkg/database/repositories/organization_users/repo_test.go`
- Modified: `services/billing/api/*_test.go` (6 files — updated JWT call signatures)
- Modified: `services/authentication/api/init_test.go`, `services/common/api/organizations_test.go`

---

## Service / Business Logic

- [x] New or changed service methods follow existing patterns
- [ ] Transactions used appropriately for multi-repository updates
- [x] Error handling uses common error codes and translations
- [ ] Cross-service impacts considered

Notes:
- `SignIn` and `ChooseOrganization` now pass org name and logo to `GenerateAuthenticationToken`
- Repository `GetOrganizationUserWithOrganization` preloads `logo` field in addition to existing fields
- Seed engine gains `buildInlineSubquery` for nested query references in YAML where clauses

---

## Database / Data

- [x] New migrations added and ordered correctly
- [ ] Migrations tested up/down locally
- [x] Seeders / mock data adjusted if needed
- [ ] Data backfill / cleanup plan documented if needed

Migration 000225 down: adds cleanup of `subscription_plan_version_features` before feature deletion (FK is ON DELETE RESTRICT).
Mock data: all feature queries now include `platform_id` filter via inline subqueries.

---

## Deployment / Rollback

- **Target env**: dev
- **Config / env vars**: none
- **Data / migrations**: 1 modified migration down script
- **Rollback**: Revert merge commit

---

## Quick Checklist

- [ ] Code formatted (gofmt / goimports, linters clean) & no stray debug logs
- [ ] Logs and errors use common helpers & error codes where relevant
- [ ] Docs / OpenAPI / Postman collections updated if relevant
- [ ] Reviewers + labels set
```

---

## Example 2: Bug Fix with Auto-Detected Jira from Branch

**Invocation**: `/create-pr`

**Branch**: `fix/IDX-789-duplicate-key-role-update`

The skill extracts `IDX-789` from the branch name automatically. No prompt to the user.

**Commits**:
```
x1y2z3a fix(rbac): resolve duplicate key error on role update
b4c5d6e test(rbac): add regression test for role update conflict
```

**Generated title**: `IDX-789: fix(rbac): resolve duplicate key error on role update`

**Key differences from Example 1**:
- Only "Bug fix" checked in Type of Change
- Scope: "RBAC service, user roles repository"
- Database / Data section: all unchecked (no migrations)
- Deployment / Rollback: "Revert merge commit"

---

## Example 3: Infrastructure-Only Draft PR

**Invocation**: `/create-pr --draft`

**Branch**: `chore/upgrade-redis-7.2`

No Jira tickets found in branch name or commits. The skill prompts:
> "No Jira tickets detected. Please provide Jira ticket ID(s) (e.g., IDX-123), or type 'none' to skip."

User responds: `none`

**Generated title**: `chore(infra): upgrade Redis to 7.2`

**Key sections**:

```markdown
## Type of Change

- [ ] Feature
- [ ] Bug fix
- [ ] Refactor / cleanup
- [x] Configuration / infra
- [ ] Database / migrations
- [ ] Breaking change

---

## How It Was Tested

- **Automated**:
  - [ ] Unit tests
  - [ ] Integration / e2e / service tests
  - [x] Not applicable (infrastructure-only change, no application code modified)

---

## Service / Business Logic

N/A - infrastructure only change.

---

## API / Contracts

N/A - no API changes.

---

## Database / Data

N/A - no database changes.

---

## Deployment / Rollback

- **Target env**: dev
- **Config / env vars**: Updated Redis image tag in docker-compose.yaml
- **Data / migrations**: none
- **Rollback**: Revert Redis image tag to previous version and redeploy
```

PR is created with `--draft` flag.

---

## Example 4: Multi-Layer Feature

**Invocation**: `/create-pr IDX-100`

**Branch**: `feat/IDX-100-device-host-crud`

Changes span all layers: migrations, schemas, repositories, services, handlers, OpenAPI, mock data, Kong routes, tests.

**Generated title**: `IDX-100: feat(ftm): add device host CRUD endpoints`

**Key differences**:
- All applicable Type of Change boxes checked: Feature + Database/migrations
- All Service / Business Logic boxes checked (spans multiple services, uses transactions)
- API / Contracts: OpenAPI updated `[x]`, backward compatible `[x]`, error codes aligned `[x]`
- Database / Data: migrations `[x]`, seeders `[x]`
- Scope lists many layers: "migrations, schemas, repositories, FTM service, FTM handlers, OpenAPI, mock data, Kong routing"
- Deployment section lists new env vars if any were added
- Rollback: "Rollback migrations: `make -C infra/database down n=2`. Revert merge commit."
