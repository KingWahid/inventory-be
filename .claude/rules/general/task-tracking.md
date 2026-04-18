# Task Tracking

**Always use the todo list tools** (`TodoWrite`) for every non-trivial task. Use todos when planning multi-step work so progress is visible to the user.

## When to Create Tasks

- After receiving a multi-step request — capture it as tasks before starting work
- When discovering sub-tasks during implementation — add them as new entries
- When a delegated agent identifies follow-up work — create the task, don't just mention it

Skip todos for single-step trivial work (one-line fix, rename, answer a question).

## Task Subject Convention

Format: `{verb} {what} {where}`

- **Verb**: imperative form — `Create`, `Add`, `Fix`, `Update`, `Remove`, `Refactor`, `Wire`, `Test`
- **What**: the thing being changed — concise but specific
- **Where**: the layer or module, in parentheses when helpful for clarity

Agent assignment (if delegating) goes in the task **description**, not the subject. The subject stays readable as a short verb phrase.

### Examples

| Good | Bad |
|------|-----|
| `Create user schema and migration` | `user stuff` |
| `Add rate limiting to signin route (Kong)` | `Kong changes` |
| `Fix N+1 query in ListDevices (repository)` | `fix performance issue` |
| `Wire subscription consumer (worker)` | `consumer wiring` |
| `Update OpenAPI spec for billing endpoints` | `update spec` |
| `Test organization scoping in GetUser` | `add tests` |

## activeForm Convention

Present continuous form of the subject — shown in the spinner while in progress.

| Subject | activeForm |
|---------|------------|
| `Create user schema and migration` | `Creating user schema and migration` |
| `Fix N+1 query in ListDevices` | `Fixing N+1 query in ListDevices` |
| `Test organization scoping in GetUser` | `Testing organization scoping in GetUser` |

## Task Lifecycle

1. **Create** — add task with subject and activeForm
2. **Start** — status to `in_progress` BEFORE beginning work
3. **Complete** — status to `completed` AFTER verifying the work is done
4. **Discover** — if new sub-tasks emerge, add them immediately

Never leave a task in `in_progress` when moving to something else — either complete it or create a blocking task explaining why it's paused.

## Description Content

The description must contain enough context for any agent to execute the task without asking follow-up questions:

- What needs to change and why
- Which files or modules are involved
- Acceptance criteria — how to know it's done
- Dependencies — what must be done first
- If delegating: which agent (`db-dev`, `biz-dev`, `app-dev`, `infra-dev`) will receive this task
