---
name: address-pr-comments
description: Triage, fix, and respond to PR review comments. Analyzes each comment against the codebase, presents verdicts for user approval, then applies fixes, commits, pushes, and replies on GitHub. Use when addressing PR review feedback.
---

# Address PR Comments

Analyze PR review comments, verify their validity against the codebase, present verdicts for user approval, then fix/dismiss and reply on GitHub.

## Trigger

Use this skill when:
- User asks to "address comments", "fix PR feedback", or "handle review comments"
- User shares PR comment URLs to address
- User says "address these", "fix these comments", or "respond to review"

## Prerequisites

- [ ] `gh` CLI is authenticated (`gh auth status`)
- [ ] Current branch has an open PR, or user provides PR number/comment URLs

## Process

### 1. Detect PR and Gather Comments

**If user provided specific comment URLs:**
Extract comment IDs from the URL pattern `#discussion_r{id}`. Determine the PR number from the URL path (`/pull/{number}`).

```bash
# Extract PR info from current branch
gh pr view --json number,url,headRefName
```

**If no URLs provided (all unresolved mode):**
Fetch all review comments for the current branch's PR:

```bash
# Get all review comments on the PR
gh api repos/{owner}/{repo}/pulls/{pr_number}/comments
```

**Error handling**:
- If no PR exists for the current branch: ask the user for a PR number or URL.
- If `gh auth status` fails: stop and instruct `gh auth login`.

### 2. Parse Comment Data

For each comment, extract:
- **id**: comment ID (for replying)
- **path**: file path the comment references
- **line** / **start_line**: line range in the file
- **body**: the reviewer's comment text
- **diff_hunk**: the code context shown in the review

If user provided specific URLs, filter to only those comment IDs.

If fetching all comments, filter out:
- Comments that are already replies (have `in_reply_to_id` set)
- Bot comments (unless they contain actionable suggestions)

### 3. Analyze Each Comment

For each comment, perform codebase verification:

1. **Read the referenced file** at the commented lines to understand the current code.
2. **Understand the suggestion**: What is the reviewer asking to change?
3. **Verify claims**: If the comment claims something exists elsewhere (e.g., "this utility already exists in X"), search the codebase to confirm.
4. **Check feasibility**: Can the suggestion be applied without breaking other code? Check for usages of the current code.
5. **Classify the verdict**:
   - **Fix**: Comment is valid and actionable. Draft the fix approach.
   - **Dismiss**: Comment is a false positive, over-engineering, or not applicable. Draft dismissal reasoning.
   - **Discuss**: Subjective or unclear — needs human judgment. Note what's unclear.

### 4. Present Verdicts

Display a summary table of all comments with verdicts:

```
| # | File:Line | Verdict | Summary |
|---|-----------|---------|---------|
| 1 | pkg/services/auth/service.go:18 | Fix | Duplicate utility — use shared one from pkg/common |
| 2 | workers/tasks/export.go:21 | Dismiss | Current approach is idiomatic Go |
| 3 | services/billing/api/handler.go:45 | Discuss | UX concern — needs product input |
```

Then for each comment, show:
- The original comment (abbreviated)
- The verdict with detailed reasoning
- For **Fix**: the planned change
- For **Dismiss**: the reply that will be posted
- For **Discuss**: what needs clarification

**Ask the user to confirm or override each verdict.** Wait for explicit approval before proceeding. The user may:
- Approve all verdicts as-is
- Override a "Fix" to "Dismiss" (or vice versa)
- Provide additional context for "Discuss" items
- Edit the planned fix or reply text

### 5. Apply Fixes

For each confirmed **Fix** verdict:
- Edit the referenced file with the planned change
- Verify the fix doesn't introduce compilation errors or break imports

For **Dismiss** and **Discuss** verdicts: no code changes.

After all fixes are applied, run a quick sanity check:
```bash
go build ./...
```

If the build fails, fix the issue before proceeding.

### 6. Commit, Push, and Reply

**Commit and push:**
```bash
# Stage all changed files
git add {changed_files}

# Commit with descriptive message
git commit -m "fix: address PR review comments

- {summary of each fix applied}

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"

# Push to PR branch
git push
```

**Reply to each comment on GitHub:**

For **Fixed** comments:
```bash
gh api repos/{owner}/{repo}/pulls/{pr}/comments \
  -X POST \
  -f body="Fixed — {explanation of what was changed}." \
  -F in_reply_to={comment_id}
```

For **Dismissed** comments:
```bash
gh api repos/{owner}/{repo}/pulls/{pr}/comments \
  -X POST \
  -f body="{reasoning for not changing}." \
  -F in_reply_to={comment_id}
```

For **Discuss** comments:
```bash
gh api repos/{owner}/{repo}/pulls/{pr}/comments \
  -X POST \
  -f body="{question or clarification needed}." \
  -F in_reply_to={comment_id}
```

### 7. Report

Provide a summary:
- How many comments: fixed / dismissed / discussed
- Commit hash of the fix
- PR URL
- Any remaining items that need attention

## Error Handling Summary

| Error | Recovery |
|-------|----------|
| No PR for current branch | Ask user for PR number or URL. |
| `gh` CLI not authenticated | Stop. Instruct: `gh auth login`. |
| Comment URL parse failure | Skip with warning, continue with others. |
| Referenced file no longer exists | Flag as "Discuss" — code may have been refactored. |
| No unresolved comments found | Inform user, stop. |
| Build fails after fixes | Fix compilation errors before committing. |
| `gh api` reply fails | Show error, continue with remaining replies. |

## Checklist

- [ ] PR detected (from branch or user input)
- [ ] Comments fetched (specific URLs or all unresolved)
- [ ] Each comment analyzed against actual codebase
- [ ] Verdicts presented to user with reasoning
- [ ] User approved/overridden verdicts before any changes
- [ ] Fixes applied for confirmed "Fix" verdicts
- [ ] `go build ./...` passes after fixes
- [ ] Changes committed and pushed
- [ ] Replies posted to all comments on GitHub
- [ ] Summary report provided
