# CLAUDE.md

Guidance for Claude Code when working in this repo. Applies on top of (not instead of) the
`/ideate` → `/implement` → `/review` skills in `.claude/skills/` and the agents in `.claude/agents/`.

## Workflow

- **`/implement` auto-chains to `/review`.** When the user runs `/implement`, run `/review` on the
  result automatically once the build gate is green — don't just recommend it and stop. Skip this
  chaining only if the user explicitly says not to review this time.
- **Always end `/implement` and `/review` with a files-changed/files-reviewed list** — a flat list of
  paths is enough; the user wants to skim it, not re-read the diff.
- **Prefer small, independently-mergeable, dependency-ordered PRs** over one large PR. When a task
  naturally splits into pieces with a dependency order, say so explicitly in the `/implement` report
  and suggest the split — don't silently bundle them:
  - A DB schema change (e.g. a new column/table in `backend/*/repository.go`'s `InitDB`) is its own PR,
    and goes first if anything else depends on it.
  - A `web-server` API/response-shape change is its own PR, and goes first if `app/train_dashboard`
    needs it.
  - The task-breakdown summary on the `BACKLOG.md` entry (see below) is often the fastest way to see
    the natural split — one line in it is frequently one PR, though not always.

## `/ideate`: add a task-breakdown summary

Whenever `/ideate` drafts a new `BACKLOG.md` bullet or refines an existing one, append a short
**task-breakdown summary**: one line per affected area, naming the main change needed there. Areas are
`scraper` / `web-server` / `shared` / `app/train_dashboard` — only list the ones actually touched.
Keep each line to the essential change, not a plan:

```
- **[area] Title.** ...the usual bullet...
  - scraper: add `X` field to the fetch/filter step.
  - web-server: expose `X` on the `/summary` response.
  - app: render `X` on the trip card.
```

This is intentionally small — a few words per area, not a task list. It's folded into `/ideate`'s
normal output; there is no separate `/task-breakdown` stage in this project's pipeline.

## Code style: comments

Write comments the way someone who already knows this codebase would — not an onboarding narrator.
**If the code already shows it, don't also say it in a comment.** A comment earns its place only when
it carries a *why* the code can't express on its own.

- **Don't restate what the code obviously does.**
  ```go
  // bad — the next line already says this
  // Log and continue so one flaky station doesn't abort the whole run
  fmt.Printf("error fetching trips for station %s: %v", station.Code, err)
  continue
  ```
  ```go
  // good — "continue" is visibly not "return"; the non-obvious part is *why* that's safe
  // CP's API is flaky per-station; don't let one bad station abort the whole run.
  fmt.Printf("error fetching trips for station %s: %v", station.Code, err)
  continue
  ```
- **One-liners over multi-line.** A multi-line comment is the exception, only when the *why* genuinely
  needs it (e.g. explaining the midnight-merge logic in `scraper/repository.go`), not the default.
- **No comments about a service this code doesn't depend on.** `scraper` writing to the DB has no
  comment about how `web-server` later reads it — that's a one-way, no-dependency relationship. If
  `web-server` calls into `shared` directly, a comment there referencing `shared`'s contract is fine —
  that's a real dependency.
- **No comments about the past or the future**, except a real `// TODO` tied to a specific planned
  follow-up the user actually asked for:
  ```go
  // bad
  id string // for now we don't set this; eventually CP may give us a stable id
  // good
  id string // CP doesn't provide a stable id for this; we derive one, see resolveArrivalDay
  ```
  Drop "for now", "currently", "at the moment" qualifiers, and drop "this used to work differently"
  history — state the current fact only.

## Code style: avoid deep indentation

Default to the happy path at the outer indentation level; handle the error/exceptional case in an
early-return `if` block. This is already idiomatic Go — lean into it rather than nesting the success
case inside an `if`/`else`:

```go
// bad
if trips, err := cpClient.FetchTrips(ctx, station.Code, oneHourAgo); err == nil {
    // happy path buried a level deep
} else {
    // handle error
}

// good
trips, err := cpClient.FetchTrips(ctx, station.Code, oneHourAgo)
if err != nil {
    // handle error, return/continue
}
// happy path at the outer level
```

## Logging

- **Don't log the success/happy path** unless explicitly asked to. A job-completion heartbeat
  (`"scrape finished successfully"`) is fine — that's a run-level status, not a per-item success log.
- **Always log at error level when an error is caught and not re-thrown/propagated** — never swallow an
  error silently. (This is also something `review-correctness`/`review-go` check for — catch it before
  review, not after.)

## Metrics

Not yet applicable — this project has no metrics/monitoring system. If one gets added later:
- don't add a custom metric unless explicitly asked to;
- never put a high-cardinality value (a trip id, a train number) in a metric label/dimension — an enum
  or bounded category is fine, an id is not.
