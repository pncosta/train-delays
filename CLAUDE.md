# CLAUDE.md

Guidance for Claude Code when working in this repo. Applies on top of (not instead of) the
`/ideate` → `/implement` → `/review` skills in `.claude/skills/` and the agents in `.claude/agents/`.

## Workflow

- **`/implement` auto-chains to `/review`.** When the user runs `/implement`, run `/review` on the
  result automatically once the build gate is green — don't just recommend it and stop. Skip this
  chaining only if the user explicitly says not to review this time.
- **Always end `/implement` and `/review` with a files-changed/files-reviewed list** — a flat list of
  paths is enough; the user wants to skim it, not re-read the diff.
- **`/review` auto-fixes cheap findings, no need to ask first.** A finding is cheap when the fix is
  small, localized, and unambiguous (a copy/wording correction, a missing null/empty fallback, a
  contrast/color tweak, a missing event handler, dead-code removal, deduplicating into an existing
  helper) — severity label doesn't decide this, a MEDIUM finding can still be cheap. Send cheap findings
  straight back to the `implementer`, re-verify the build, then report what was found and fixed. Only
  ask first when a finding implies a real scope/design decision or the fix direction is genuinely
  ambiguous.
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

## Code style: readability and simplicity

Optimize for a person reading and debugging the code, not for fewer characters or a faster run —
nothing in this project is time-critical.

- **Descriptive names.** No single-letter names except loop counters (`i`, `j`, `k`), and no
  non-standard abbreviations: `delay` not `d`, `stats` not `s`, `threshold` not `thr`. Standard ones
  (`ctx`, `db`, `err`, `id`, `URL`) are fine.
- **Plain control flow.** Simple loops and `if`s with named intermediate variables beat dense
  one-liners. No multi-clause `for` headers doing date arithmetic, no `max`/`min` tricks on strings, no
  function returning an unexplained `(value, bool)` — name the check instead.
- **Simplicity over performance.** Prefer the version that's easiest to read and debug even if it does
  a few more round trips or loops. Optimize only once a real cost or timeout problem shows up (Turso
  bills rows read, not queries).
- **Don't over-engineer.** Build for how the code actually runs. Check what the platform already
  provides (Cloud Run job retries, Cloud Scheduler) before adding markers, resumability, batching or
  retry logic, and don't handle cases that can't happen. Start with the plainest version that works and
  add complexity only when a real need shows up.

```go
// bad
for s := time.Date(f.Year(), f.Month(), 1, 0, 0, 0, 0, time.UTC); s.Format(l) < u; s = s.AddDate(0, 1, 0) {

// good
day := firstTripDate
for day.Format(dayLayout) < today {
	dayString := day.Format(dayLayout)
	// ...
	day = day.AddDate(0, 0, 1)
}
```

## Code structure: repository / service / handlers

- **`repository.go`** is a thin facade over SQL: one function per basic operation (select, insert;
  a transaction that only runs writes is fine), SQL written inline and plain (no CTEs, subqueries, window
  functions, or CASE-based rules), with no business rules, thresholds, aggregation, or loops beyond
  scanning/writing. Rows are scanned into plain Go types (no `sql.Null*` leaking out).
- Jobs never delete rows to make re-runs work: they insert with `ON CONFLICT DO NOTHING` (an existing
  row means it was already processed). Recomputing is a manual delete.
- Structs for well-known entities used across files (e.g. a `Widget`) go in **`models.go`**; a type only
  one file uses stays in that file. Helper methods encoding business rules on them go in `service.go`, and
  the repository never calls them.
- **`service.go`** holds the business logic as plain functions, not methods on a struct. Each takes
  everything it needs as parameters (`ctx`, the `*sql.DB`, `now`, ...), calls repository functions, and
  does the computing and looping in Go, e.g. `summarizeWidgets(ctx, db, day)` calling `selectWidgets`.
- **`handlers.go`** (HTTP services) parses/validates the request, calls the service, and maps errors to
  responses. For a job, `main.go` plays this role: read env, open the DB once, call schema init (e.g.
  `initSchema`) and the service directly as wiring, exit non-zero on failure.
- Repeating a few lines is fine; don't add a type, helper or generic whose only job is to avoid that
  repetition (e.g. a struct embedded in two others just to share two fields).
- Applies to new code and to `rollup`; `scraper` and `web-server` predate it — migrate them only when asked.

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
