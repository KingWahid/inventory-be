# Planning (Plan Mode)

When entering plan mode, follow this structure. Plans must be concrete enough that any domain agent (`db-dev`, `biz-dev`, `app-dev`, `infra-dev`) can execute their portion without guessing.

## When to Enter Plan Mode

**Default: enter plan mode for multi-layer or design-heavy work.** For obvious single-layer fixes, code directly without plan mode.

### Skip plan mode when ALL of these are true:

- Single, isolated change (typo, rename, small bug fix, config tweak)
- Touches **≤3 files**
- **No design decisions** — one reasonable approach
- **No new files** (except test files for existing code)
- Stays within **one domain** (only database, only service, etc.)

### Examples

| Task | Plan mode? | Why |
|------|-----------|-----|
| Fix typo in error message | No | One file, obvious fix |
| Add a new CRUD entity | **Yes** | Multiple layers, design decisions, new files |
| Update an existing service method | **Yes** | May affect tests, consumers, handlers |
| Add a field to a schema | **Yes** | Migration + schema + repository + possibly service/handler |
| Rename a variable | No | One file, trivial |
| Add a new consumer for event X | **Yes** | Multiple files, wiring decisions, test expectations |
| Fix a failing test | No | Usually one file, obvious fix |
| Add rate limiting to an endpoint | **Yes** | Kong config + possible service changes, design decision |
| Update a log message | No | One file, trivial |
| Implement a new background process | **Yes** | Multiple layers, task type registration, handler, tests |

### When in doubt, enter plan mode.

The cost of planning is low. The cost of implementing the wrong thing is high (rework, context waste, bugs).

## Before Writing the Plan

1. **Read the request** — understand what the user wants, not just what they said
2. **Read `docs/conventions/codebase-conventions.md`** — refresh on project structure
3. **Explore affected areas** — read existing code in the modules that will change
4. **Find a reference implementation** — locate an existing similar feature (e.g., for a new entity CRUD, find an existing entity's migration → schema → repository → service → handler chain)
5. **Identify affected layers**:
   - **Database** (`pkg/database/`, `infra/database/`) → `db-dev`
   - **Business Logic** (`pkg/services/`, `pkg/common/`, `pkg/eventbus/`, consumer handlers) → `biz-dev`
   - **Application** (`services/`, `workers/`, `sync-services/`) → `app-dev`
   - **Kong routing** (`infra/kong/kong.template.yml`) → `biz-dev`
   - **Infrastructure** (Dockerfiles, docker-compose, `.air.toml`, entrypoints, CI workflows, `.env.*`) → `infra-dev`
   - **Database migrations** (`infra/database/`) → `db-dev` (NOT `infra-dev`)

## Plan Structure

Every plan must include these sections in order:

### 1. Context

2-3 sentences: what is being built/changed and why. Reference the user's request.

### 2. Reference Implementation

Point to an existing similar feature. Include specific file paths for each layer. The implementation should follow this reference's patterns (naming, structure, error handling, FX wiring).

Example:
```
Reference: Device entity
- Migration: infra/database/migrations/000042_create_devices.up.sql
- Schema: pkg/database/schemas/device.go
- Repository: pkg/database/repositories/device/repository.go
- Service: pkg/services/device/service.go
- Handler: services/ftm/api/handler_device.go
- Converter: services/ftm/api/converter_device.go
```

If no similar feature exists, state that explicitly and list the patterns to follow from the rules.

### 3. Changes

List every file that will be created or modified, grouped by agent assignment:

```
**db-dev (Database Layer)**
- `infra/database/migrations/NNNNNN_create_xxx.up.sql` — Create
- `infra/database/migrations/NNNNNN_create_xxx.down.sql` — Create
- `pkg/database/schemas/xxx.go` — Create
- `pkg/database/repositories/xxx/repository.go` — Create (interface + implementation)
- `pkg/database/repositories/xxx/mocks/Repository.go` — Generate (mockery)

**biz-dev (Business Logic Layer)**
- `pkg/services/xxx/types/types.go` — Create (domain types)
- `pkg/services/xxx/service.go` — Create (interface + implementation)
- `pkg/services/xxx/fx.go` — Create (FX module)
- `infra/kong/kong.template.yml` — Modify (add route if new endpoint needs special auth/rate limiting)

**app-dev (Application Layer)**
- `services/yyy/api/handler_xxx.go` — Create
- `services/yyy/api/converter_xxx.go` — Create (FromStub/ToStub converters)
- `services/yyy/api/openapi.yaml` — Modify (add endpoints)
```

For each file:
- **Action**: Create / Modify / Delete / Generate
- **Purpose**: what changes and why (one line), omit if obvious from file name

### 4. Task Breakdown

Ordered tasks following the task naming convention (`{verb} {what} {where}`), with agent assignment and dependencies:

```
1. Create X table migration (db-dev)
2. Create X schema and repository (db-dev, blocked by: 1)
3. Generate X repository mock (db-dev, blocked by: 2)
4. Create X service with CRUD operations (biz-dev, blocked by: 3)
5. Create X handler, converter, and OpenAPI spec (app-dev, blocked by: 4)
6. Add Kong route for X (biz-dev, blocked by: 5, if needed)
```

Layer execution order: database → business logic → application (bottom-up). Tasks within the same layer can run in parallel if they don't touch the same files.

### 5. Key Decisions

List design decisions with multiple valid approaches:
- What the decision is
- Which approach this plan takes
- Why (1 sentence)

Open questions needing user input go here — use `AskUserQuestion` before finalizing.

Common decisions to surface:
- Caching strategy (cache-aside? which keys? TTL?)
- Event-driven vs synchronous
- Organization-scoped vs unscoped
- Soft delete vs hard delete
- Which error codes to use (reference `pkg/common/errorcodes/`)

### 6. Verification

Concrete verification steps:

```
- [ ] `go build ./...` passes
- [ ] Migration applies cleanly (if new migrations)
- [ ] Mocks regenerated if repository interfaces changed: `make mocks-generate`
- [ ] `make test-all` passes (if dev containers are up) OR `make test` passes (unit tests only)
- [ ] Run all 4 review agents (`correctness-reviewer`, `security-reviewer`, `quality-reviewer`, `rules-reviewer`) on modified files
```

**Test suite rule:** If dev containers are running, use `make test-all` (includes integration tests). If containers are NOT running, fall back to `make test` (unit tests only). Never skip the test suite entirely.

### 7. Manual Testing

**Required in every plan.** The lead performs manual testing after implementation + verification pipeline + code review complete. Runs only if dev containers are up.

List concrete test cases executed by CLI (`curl`, `mosquitto_pub`/`mosquitto_sub`, `redis-cli`, `psql`). Tests use **existing mock data** from `infra/database/cmd/seed/mock-data/`.

#### What to Include

For each test case:
- **ID** — sequential (MT-1, MT-2, ...)
- **Description** — what is being tested (one line)
- **Precondition** — required mock data or state (reference specific mock data files/records)
- **Command** — exact CLI command
- **Expected result** — HTTP status code, response body shape, side effects (DB record created, event published)

#### Example

```
**MT-1: Create device host**
- Precondition: Organization from mock data (org_id from mock-data/organizations.go)
- Command: curl -X POST http://localhost:8080/device-hosts \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "X-Organization-Id: <org_id>" \
    -d '{"name": "Test Host", "location": "Floor 1"}'
- Expected: 201 Created, response contains id, name, location

**MT-2: List device hosts**
- Precondition: MT-1 completed (device host exists)
- Command: curl http://localhost:8080/device-hosts \
    -H "Authorization: Bearer $TOKEN" \
    -H "X-Organization-Id: <org_id>"
- Expected: 200 OK, array contains the device host from MT-1

**MT-3: Create device host with duplicate name**
- Precondition: MT-1 completed
- Command: curl -X POST http://localhost:8080/device-hosts \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "X-Organization-Id: <org_id>" \
    -d '{"name": "Test Host", "location": "Floor 2"}'
- Expected: 409 Conflict, error_code for duplicate
```

#### Test Case Coverage

Plans must include test cases for:
- **Happy path** — each new endpoint or feature works with valid input
- **Error cases** — invalid input, missing required fields, duplicate records, not-found scenarios
- **Authorization** — requests without token return 401, wrong org returns 403/404
- **Edge cases** — empty lists, boundary values, concurrent operations (if relevant)

For MQTT features, test publish/subscribe flows. For event-driven features, verify events are published and consumed (check outbox table or consumer side effects).

#### Obtaining Auth Tokens

To get a JWT token for manual testing, sign in with mock user credentials:
```
curl -X POST http://localhost:8081/users/signin \
  -H "Content-Type: application/json" \
  -d '{"email": "<mock_user_email>", "password": "<mock_user_password>"}'
```
Reference the mock user data in `infra/database/cmd/seed/mock-data/` for credentials.

## Plan Review (Before Presenting to User)

**When:** After writing all 7 sections. **Before:** Calling `ExitPlanMode`.

The plan must pass review by 2 specialized agents before being presented to the user.

### How to Run

Spawn both plan review agents **in parallel** using a single message with 2 `Agent` tool calls. Each must specify the plan file path. The caller decides the model.

```
Agent(subagent_type="plan-design-reviewer", model="sonnet",
      prompt="Review this plan for structure, conventions, and architecture: <plan_file_path>")

Agent(subagent_type="plan-execution-reviewer", model="sonnet",
      prompt="Review this plan for executability and test coverage: <plan_file_path>")
```

### Plan Review Agents

| Agent | Focus | Catches |
|-------|-------|---------|
| `plan-design-reviewer` | Structure + convention + architecture | Missing sections, vague steps, wrong naming, incorrect patterns, layer violations, wrong agent assignments, missing org scoping |
| `plan-execution-reviewer` | Executability + test coverage | Undefined types, missing signatures, vague descriptions, handoff gaps, missing error/auth/edge case scenarios, incomplete test expectations |

Agent definitions live in `.claude/agents/plan-*.md`.

### Handling Findings

After both agents return:

1. **Collect all findings** — group by severity (blocking vs should-fix)
2. **Fix all blocking findings** — update the plan directly:
   - Structural gaps (missing sections, vague steps)
   - Convention violations (wrong naming, incorrect patterns)
   - Boundary violations (cross-layer tasks, wrong agent assignments)
   - Security concerns (missing org scoping, auth gaps)
   - Missing test scenarios (no error cases, no auth tests)
   - Blocking clarity gaps (undefined types, missing signatures, vague handoff points)
3. **Address should-fix findings** — apply judgment. If unsure, include them.
4. **Re-run if significant changes were made** — if fixing findings changed the plan's structure substantially (new tasks, reorganized dependencies, major rewrites), run the 2 reviewers again. Minor wording fixes do not require re-review.
5. **Only call `ExitPlanMode` when all blocking findings are resolved.**

### Skipping Plan Review

Plan review can be skipped **only** when:
- The plan is a trivial update to an existing plan (e.g., marking completed tasks during mid-execution changes)
- The user explicitly asks to skip review

In all other cases, plan review is mandatory.

## New Requests During Implementation

**The plan is the single source of truth for the current task.** Every new user request must either be reflected in the plan or explicitly determined to not need one.

### Decision Flow

When the user sends a new request while implementation is in progress:

1. **Evaluate** against the skip criteria in "When to Enter Plan Mode"
2. **If it qualifies for skipping** (one-off fix, ≤3 files, no design decisions, no new files, single layer) — do it directly and continue with the existing plan
3. **If it does NOT qualify** — go back to plan mode. No exceptions.

### Going Back to Plan Mode

1. **Stop current work**
2. **Enter plan mode** — call `EnterPlanMode`
3. **Update the existing plan** — do NOT start from scratch. Modify affected sections:
   - **Context** — append the new request
   - **Changes** — add/remove/modify files as needed
   - **Task Breakdown** — add new tasks, update dependencies, mark completed tasks (e.g., `1. ~~Create X migration~~ ✅`)
   - **Key Decisions** — add new decisions if the request introduces trade-offs
   - **Manual Testing** — add new test cases for the new behavior
4. **Include current state** — document what has already been done (completed tasks, created files, generated mocks)
5. **Run plan review** — the updated plan goes through the same 2 plan review agents. Skip only if truly trivial.
6. **Get approval** — call `ExitPlanMode` and wait for user approval before resuming

### Why This Matters

Without this discipline:
- Agents work from a stale plan that no longer reflects reality
- New tasks get implemented without test cases or verification steps
- Dependencies break because new work wasn't properly sequenced
- The user loses visibility into what's being done and why

### Examples

| User request during implementation | Action |
|-------------------------------------|--------|
| "Actually, also add soft delete to that entity" | **Plan mode** — new migration, schema change, service logic, test cases |
| "Fix the typo in that error message you just wrote" | **Direct fix** — one file, obvious change |
| "Add an event when this entity is created" | **Plan mode** — event definition, publisher, consumer, test cases |
| "Use `float64` instead of `float32` there" | **Direct fix** — one file, convention compliance |
| "Also expose this via the API" | **Plan mode** — handler, converter, OpenAPI spec, Kong route, test cases |
| "Rename that method to something clearer" | **Direct fix** — single rename, no design decision |

## Rules

- **No vague steps** — "implement the service" is not a plan step. "Create DeviceService with CreateDevice, GetDevice, ListDevices methods that call DeviceRepository" is.
- **No missing files** — every file that will be touched must appear in Changes. If unsure, explore first.
- **No assumptions about schema** — specify column names, types, constraints, indexes, and foreign keys.
- **Reference over invention** — always point to an existing feature. Copy the structure, adapt the content.
- **One concern per task** — a task should not span layers (e.g., "create migration AND service" is two tasks).
- **Dependencies must be explicit** — use `blocked by: N` format.
- **Review is part of the plan** — verification section must include running the 4 review agents.
- **Manual testing is part of the plan** — every plan must list concrete manual test cases in section 7.
- **Plan review before presenting** — run 2 plan review agents (design + execution) before calling `ExitPlanMode`.
