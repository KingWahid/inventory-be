# Explore Agent Rules

## The Rule

**Keep Explore agent scope narrow and specific.** Never send an Explore agent on a broad, open-ended search. Each invocation should target a single, well-defined question.

## Why

Broad exploration wastes tokens, returns noisy results, and often misses the specific answer. A focused query finds what you need faster and stays within context limits.

## Good vs Bad Prompts

| Bad (too broad) | Good (focused) |
|-----------------|----------------|
| "Explore the database layer and understand how repositories work" | "Find how `DeleteDevice` in `pkg/database/repositories/device/` handles soft delete" |
| "Look at the services directory and summarize the patterns" | "Find how `pkg/services/device/service.go` calls the repository in `CreateDevice`" |
| "Understand how authentication works in this codebase" | "Find where JWT validation middleware is applied in `services/iam/`" |
| "Explore the worker infrastructure" | "Find how consumers are registered in `workers/jobs/consumers/` — show the wiring pattern" |
| "Search for all error handling patterns" | "Find how `db_utils.ClassifyDBError` is used in `pkg/database/repositories/device/`" |

## Guidelines

1. **One question per agent** — if you have 3 questions, spawn 3 agents in parallel, each with a tight scope
2. **Name the files or directories** — always include the specific path or module the agent should look in
3. **State what you're looking for** — "find the pattern for X", "locate where Y is defined", "show how Z is wired"
4. **Set thoroughness appropriately** — use "quick" for known-location lookups, "medium" for pattern discovery in a module, "very thorough" only when searching across multiple unrelated directories
5. **Prefer Glob/Grep first** — if you know the file name or a unique string, use the direct tools. Only use Explore when the answer requires reading and understanding multiple files
