# Backlog

Dumping ground for ideas, known bugs, and improvements for pt-train-delays. Not a sprint board — just so nothing good gets lost. Add freely; prune when stale.

## Bugs

- **[scraper] One failed station aborts the whole run.** `getAndStoreTrips` returns immediately on the first station's API error, skipping every other station for that cycle. CP's API is flaky, so this likely loses more data than the midnight bug. Should log-and-continue per station instead.
- **[scraper] DB connection opened/closed per insert batch.** `InsertEndingTrips`/`InsertStartingTrips` in `backend/scraper/repository.go` call `sql.Open` + `defer Close()` every time, ~2x per station per run (~100/run). Same class of issue as the HTTP client reuse fix already shipped for the CP API client (`a2bf6ae`) — just not applied to the DB side.
- **[scraper] Unchecked error from `http.NewRequestWithContext`** in `cp_client.go` — if it ever errors, the following `req.Header.Set` calls would panic on a nil `req`. Low likelihood, cheap to fix.

## Infra / Cost

- **Turso is billed per row read**, and the planned app usage pattern (every app-open triggers many reads) would get expensive fast. Need an aggregation/caching layer in the web-server before real users show up. (Flagged by Pedro as a known, deliberately-deferred issue.)

## Ideas / Features (unscoped)

- _(nothing logged yet — this is where ideator-agent output and brainstorms land)_

## Decisions

- Pushing/committing always stays a human-confirmed step in the main session, never delegated to an autonomous agent.

## Fixed

- **[scraper] Midnight date-reconstruction + split-row bugs.** Both fixed together: `resolveClockTime` (in `utils.go`) now picks whichever of {yesterday, today, tomorrow} + CP's `"HH:MM"` clock string lands closest to `now`, instead of always assuming today's date. The row ID is now derived per-trip instead of once per run; when a trip's arrival is captured in the early morning (`isEarlyMorning`, hour < 6 — one-sided, since a late-night arrival can never predate midnight), the scraper looks up the still-open departure row for that train number (`findOpenTripID`) and reuses its ID instead of guessing, so departure and arrival merge into one row even when captured in different cron runs across midnight. Going-forward fix only — existing split/dropped rows in the DB were not backfilled.
