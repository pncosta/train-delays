---
name: review
description: Run the pt-train-delays multi-lens code review on the current change. Determines the diff, fans out to specialized reviewer subagents (review-correctness, review-engineering, and review-go when Go files changed) in parallel, then aggregates their findings into one prioritized report. Use before committing, or when the user asks to "review", "review my changes", or "run the review".
---

# /review — multi-lens code review

Orchestrate a review of the current change by fanning out to specialized reviewer **subagents** in
parallel and merging their findings into one report. This skill is the entry point; the domain
expertise lives in the agents under `.claude/agents/`.

```
Ideate ─▶ Implement ─▶ Review
(/ideate)   (/implement)  (this skill)
```

## 1. Determine the diff to review

- Default (uncommitted working change): `git status --short` then `git diff` (add `--cached` for
  staged changes).
- If the user names a revision/range, use that instead (`git diff <range>`).

Capture the list of changed files. If there are no changes, say so and stop.

## 2. Decide which reviewers to run

Always run, on any non-trivial change:
- **review-correctness** — bugs, edge cases, data integrity, security. Covers both Go and Dart.
- **review-engineering** — design/placement across `scraper`/`web-server`/`shared`/the app, Turso
  read-cost efficiency, API-contract fit with the Flutter app. Covers both Go and Dart.

Run only when relevant:
- **review-go** — only if the diff touches any `.go` file.

There is currently no dedicated Flutter/Dart idiom lens (`review-correctness`/`review-engineering`
still cover Dart changes, just not Dart-specific idiom). If Dart changes are frequent and non-trivial,
say so in your report as worth adding a `review-flutter` agent later, following the same template as
`review-go`.

If the change is trivial (formatting-only, a generated/vendored file touched as-is), say so and skip
the review rather than spawning agents needlessly.

## 3. Fan out — in parallel

Spawn the selected reviewers as subagents **concurrently** — multiple Agent tool calls in a single
message, not one after another. Give each:
- the changed-file list, and the diff (or tell it to derive it itself via `git diff`);
- the instruction to stay within its lens and report via `ReportFindings`.

Use the matching `subagent_type` (`review-correctness`, `review-engineering`, `review-go`). These
agents are read-only; they will not modify files.

## 4. Aggregate into one report

Merge the agents' findings. Don't just concatenate — deduplicate overlapping findings (two lenses
flagging the same line), and sort the combined list by severity (highest blast-radius first).
Attribute each finding to the lens that raised it.

**A dispatched lens that never returns is a coverage gap, not a pass.** If a reviewer you spawned
hasn't come back, don't silently drop it or fabricate its verdict — note it in the report as an
uncovered lens with what it was meant to check.

Present:

```
## Review summary
<one-line gate: e.g. "2 blockers — fix before commit" or "No blocking issues.">
Reviewers run: review-correctness, review-engineering[, review-go]   (skipped: review-go — no .go files changed)

## Files reviewed
path/to/file.go
...

## Findings
[severity] path/file:line — summary   (lens: review-correctness)
  - what's wrong / failing scenario
  - suggested fix

...

## Notes
<coverage gaps, anything a reviewer flagged as worth a second look>
```

Keep it tight and actionable — the point is a clear go/no-go before commit, not a wall of text.

## 5. Auto-fix cheap findings; ask about the rest

This skill never edits files itself — fixes always go back through the `implementer` agent, never
inline here. But whether that handoff needs the user's go-ahead first depends on how cheap the fix is:

- **Cheap finding → send it to `implementer` automatically, no need to ask first.** A finding is cheap
  when the fix is small, localized, and unambiguous — a copy/wording correction, a missing null/empty
  fallback, a contrast or color-value tweak, a missing event handler, removing dead code, deduplicating
  a repeated calculation into an existing shared helper. Severity label doesn't decide this — a MEDIUM
  finding can still be cheap (e.g. a mislabeled tooltip is a wording fix, not a redesign). After the
  implementer reports back, independently re-verify the build same as any other `/implement` gate, then
  tell the user what was found and fixed — report, don't ask permission retroactively either.
- **Not-cheap finding → present it and ask before fixing.** A finding that implies a scope/design
  decision, a non-obvious trade-off, a change to the approved plan, or where the right fix direction is
  genuinely ambiguous, goes to the user first, same as before.

When a review returns a mix, split it: auto-fix the cheap ones, ask only about the rest.

## Notes

- **Deliberately not the company-scale version.** No disposition routing (`fix-now` /
  `needs-plan-change` / `surface-to-human`) and no known/accepted-scope matching — both exist upstream
  to feed an autonomous multi-iteration build loop that doesn't exist in this project's pipeline. No
  token/cost telemetry — nothing external to report it to.
- **First-run caveat.** Newly-added agents/skills aren't dispatchable in the session that created them
  — reload (`/agents` or restart) before the first `/review`.
