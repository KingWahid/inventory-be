# Agent-Driven Development Workflow

## Why

Complex tasks exhaust a single agent's context window. Delegating to focused, one-shot agents keeps the main context lean while producing higher-quality work (each agent sees only what it needs). Domain-specialized prompts catch cross-layer mistakes before they happen.

## Lead Behavior

You (the main agent) are a **pragmatic lead**:

- **Code directly for small, scoped tasks** — one-off fixes, typos, single-file renames, single-layer tweaks. Don't spawn an agent to fix a typo.
- **Delegate to domain agents** for work that is:
  - Multi-file within one layer
  - Context-heavy (would consume a lot of your window)
  - Parallelizable across layers
  - Cross-cutting a concern you want independent verification on
- **Use Task/Agent tool** for research (Explore, Plan) and review subagents. These are read-only and don't need any special wrapping.
- **Never silently drop findings** — every review finding must be fixed, escalated to the user, or explicitly acknowledged.

### When to Delegate vs. Code Directly

| Task | Approach |
|------|----------|
| Fix typo in error message | Code directly |
| Rename a variable in one file | Code directly |
| Update a log message | Code directly |
| Fix a failing test | Code directly (unless test fix reveals a bigger issue) |
| Add a new CRUD entity | Delegate to `db-dev` → `biz-dev` → `app-dev` |
| Add a field to a schema | Delegate to `db-dev` (touches migration + schema + repo + possibly service/handler) |
| Add a new consumer | Delegate to `biz-dev` (logic) then `app-dev` (wiring) |
| Add rate limiting to an endpoint | Delegate to `biz-dev` (touches Kong config + possibly service) |
| Implement a background process | Delegate — multiple layers |

### Lead Responsibilities

1. Analyze the user's request and identify which layers are affected
2. Decide: code directly or delegate
3. If delegating:
   - Write a plan (see `.claude/rules/general/planning.md`)
   - Pass the plan through the 2 plan reviewers
   - Dispatch domain agents with clear task descriptions
   - Coordinate cross-agent handoffs (e.g., `biz-dev`'s consumer handler name must match what `app-dev` wires)
4. Run the **Verification Pipeline** after implementation
5. Run **Code Review** agents after verification passes
6. Run **Manual Testing** against the dev environment

## Domain Agents

Four domain-specialized agents live in `.claude/agents/`. Each is a one-shot agent spawned via the `Agent` tool with `subagent_type` matching the agent name. The caller decides the model.

### `db-dev` — Database Layer

**Owns:** `pkg/database/`, `infra/database/migrations/`, `infra/database/cmd/seed/`
**Not allowed to touch:** `pkg/services/`, `services/`, Kong config, Dockerfiles

### `biz-dev` — Business Logic Layer

**Owns:** `pkg/services/`, `pkg/common/`, `pkg/eventbus/`, consumer handler **logic** in `workers/jobs/consumers/`, `infra/kong/kong.template.yml`, translation files
**Not allowed to touch:** database layer, app-layer wiring, Dockerfiles

### `app-dev` — Application Layer

**Owns:** `services/` (handlers, converters, OpenAPI, FX), `workers/` (infrastructure and wiring, NOT consumer logic), `sync-services/`
**Not allowed to touch:** database layer, service logic, Kong routing rules, Dockerfiles

### `infra-dev` — Infrastructure Layer

**Owns:** Dockerfiles, docker-compose, `.air.toml`, entrypoint scripts, `.github/workflows/`, `.env.*`, `infra/kong/` (except `kong.template.yml`), `infra/redis/`, `infra/mosquitto/`, `infra/postgres/` (config + init, not migrations), `infra/certbot/`
**Not allowed to touch:** source code in `pkg/`, `services/`, `workers/`, database migrations, Kong routing rules

Full domain definitions live in each agent's file in `.claude/agents/`.

## Dispatching Agents

Use the `Agent` tool with `subagent_type` set to the agent name:

```
Agent(
  subagent_type="db-dev",
  description="Create devices migration + schema + repo",
  prompt="Create a new table `device_hosts` with columns... <full task description>"
)
```

**Model:** Choose per task. Default to `sonnet`. Use `opus` for complex refactors. Use `haiku` for simple scoped changes.

### Parallel Dispatch

Independent tasks across layers can be dispatched in parallel — send a single message with multiple `Agent` tool calls. Example: `db-dev` creating a migration can run in parallel with `infra-dev` updating a Dockerfile (they don't touch shared files).

**Never dispatch parallel agents that touch the same files.** `workers/jobs/consumers/` is the most common shared zone — `biz-dev` writes handler logic first, `app-dev` wires it after.

## Task Decomposition

### Layer-by-Layer with Dependencies

Decompose top-down, execute bottom-up:

```
1. db-dev:    migration → schema → repository (+ tests, mock generation)
      ↓ blocks
2. biz-dev:   service logic → event definitions → consumer handler logic (+ tests)
      ↓ blocks
3. app-dev:   HTTP handler → converter → wire consumer → OpenAPI spec (+ tests)

4. infra-dev: Dockerfiles, docker-compose, .air.toml, CI workflows (usually independent)
```

### Dependency Rules

- `biz-dev` tasks are **blocked by** `db-dev` tasks (services depend on repositories)
- `app-dev` tasks are **blocked by** `biz-dev` tasks (handlers depend on services; consumer wiring needs handler logic)
- `infra-dev` tasks are typically **independent** — can run in parallel unless the task changes how a service is built (new env vars, Dockerfile restructure)

### POC-First Strategy for Multi-Service Changes

When a change in shared packages (`pkg/`) requires updates across multiple services (`services/`, `sync-services/`, `workers/`), do **not** roll out to all services at once:

1. **Pick one POC service** — default to `services/authentication/` unless another is more relevant. Should be lightweight and fast to build/test.
2. **Implement end-to-end in the POC only** — shared packages, the POC's config/handlers/tests.
3. **Run the verification pipeline against the POC.**
4. **Report back and ask the user** whether to proceed with rollout.
5. **Roll out to remaining services in parallel** — shared package changes are already validated.

**In the plan:** Multi-service changes include a "POC Phase" and "Rollout Phase" as separate sections with an explicit user checkpoint between them.

## Quality Gates

1. **Plan review** — 2 reviewers (`plan-design-reviewer`, `plan-execution-reviewer`) before presenting the plan to the user
2. **Convention compliance** — plans follow `docs/conventions/codebase-conventions.md`
3. **Test coverage** — all new code includes tests
4. **Verification pipeline** — all 3 steps pass (see below)
5. **Code review** — all 3 code reviewers + `rules-reviewer` pass
6. **Manual testing** — lead executes test cases from the plan against the running dev environment

## Verification Pipeline

**When:** After implementation is complete. **Before:** Code review.

**Critical rule:** If a failure surfaces code written by a delegated agent, route it back to the responsible agent rather than fixing it directly — the agent that made the mistake should learn from its own fix. For code you wrote yourself, fix it yourself.

Follow `.claude/rules/general/verification-pipeline.md` (3 steps):

1. **Build + tests** — `make test-all` (if dev containers up) or `make test`
2. **Database verification** (if migrations or mock data changed) — test apply + rollback
3. **Docker verification** (if Dockerfiles or docker-compose changed) — `docker compose build` and verify startup

If any step fails, fix the root cause (delegate if appropriate) and re-run the entire pipeline. Do NOT proceed to code review until all steps pass.

## Code Review

After verification passes, run all 4 review agents — 3 code reviewers + `rules-reviewer` — **in parallel**.

### Review Agents

| Agent | Focus |
|-------|-------|
| `correctness-reviewer` | Logic correctness, edge cases, plan alignment, performance (N+1, missing indexes, cache misuse) |
| `security-reviewer` | SQL injection, org scoping, auth, data exposure, FX wiring, interfaces, layer boundaries, events, permission seeds |
| `quality-reviewer` | Conventions, naming, patterns, rule compliance, test completeness, edge case coverage |
| `rules-reviewer` | Checks if code changes made project rules/conventions stale or inaccurate |

### How to Run

Spawn all 4 reviewers **in parallel** using a single message with 4 `Agent` tool calls. The caller decides the model per agent.

```
Agent(subagent_type="correctness-reviewer", model="sonnet",
      prompt="Review these files for correctness and performance. Files: [paths]. The task: [description]")

Agent(subagent_type="security-reviewer", model="sonnet",
      prompt="Review these files for security and integration: [paths]")

Agent(subagent_type="quality-reviewer", model="sonnet",
      prompt="Review these files for conventions and test completeness: [paths]")

Agent(subagent_type="rules-reviewer", model="sonnet",
      prompt="Review whether these changes make any rules stale. Files: [paths]. Summary: [what changed]")
```

### Caller Contract

1. **Must run all 4 reviewers** — no skipping
2. **Must not ignore findings** — every finding must be:
   - **Fixed** — either directly or by re-dispatching the responsible agent
   - **Escalated** — if unsure, present to the user for decision
   - **Never silently dismissed** — disagreement requires user approval
3. **`rules-reviewer` findings:** If it reports stale rules, update the affected rule files before declaring work done. Rule updates are part of the deliverable.
4. **When to run:**
   - After implementation is complete and verification pipeline passes
   - Before declaring the feature done
   - Before creating a commit or PR

## Manual Testing

**When:** After verification pipeline + code review pass.

**Who:** The lead — not delegated agents.

**Prerequisite:** Dev containers must be running. Check:
```bash
docker ps --format "table {{.Names}}\t{{.Status}}" --filter "name=industrix-" --filter "name=service-" --filter "name=sync-service-" --filter "name=workers"
```

If dev containers are NOT running, skip manual testing and inform the user.

### Apply Migrations and Mock Data

Before executing any test case, ensure the dev database has the latest schema and data:

1. **Apply new migrations** (if any):
   ```bash
   make -C infra/database up
   ```
2. **Apply mock data** (if any were created or modified):
   ```bash
   make -C infra/database seed-mock-data
   ```

Failures → route back to `db-dev`. Do not proceed to test cases until both succeed.

### What to Test

Execute the manual test cases listed in Section 7 of the plan. Use CLI tools:

- **HTTP endpoints** → `curl` with appropriate headers (Authorization, Content-Type, X-Organization-Id)
- **MQTT features** → `mosquitto_pub` / `mosquitto_sub`
- **Database side effects** → `psql` or via API
- **Redis/cache effects** → `redis-cli`
- **Event-driven flows** → verify outbox table entries or consumer side effects

### Auth Tokens

Obtain a JWT token by signing in with mock user credentials from `infra/database/cmd/seed/mock-data/`:
```bash
curl -s -X POST http://localhost:8081/users/signin \
  -H "Content-Type: application/json" \
  -d '{"email": "<mock_email>", "password": "<mock_password>"}' | jq -r '.token'
```

### Execution Flow

1. Verify dev containers are running
2. Apply new migrations and mock data (route failures to `db-dev`)
3. Obtain auth token
4. Execute each test case (MT-1, MT-2, ...) in order
5. Compare actual vs expected for each case; log pass/fail
6. If any test fails:
   - Identify the responsible agent (or fix directly if it's your code)
   - Fix, then **re-run ALL manual tests** — fixes may break other cases
7. Report results to the user
