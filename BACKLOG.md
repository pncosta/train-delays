# Backlog

Dumping ground for ideas, known bugs, and improvements for pt-train-delays. Not a sprint board — just so nothing good gets lost. Add freely; prune when stale.

## Bugs

- **[scraper] DB connection opened/closed per insert batch.** `InsertEndingTrips`/`InsertStartingTrips` in `backend/scraper/repository.go` call `sql.Open` + `defer Close()` every time, ~2x per station per run (~100/run). Same class of issue as the HTTP client reuse fix already shipped for the CP API client (`a2bf6ae`) — just not applied to the DB side.
- **[scraper] Unchecked error from `http.NewRequestWithContext`** in `cp_client.go` — if it ever errors, the following `req.Header.Set` calls would panic on a nil `req`. Low likelihood, cheap to fix.

## Infra / Cost

- **Turso is billed per row read**, and the planned app usage pattern (every app-open triggers many reads) would get expensive fast. Need an aggregation/caching layer in the web-server before real users show up. (Flagged by Pedro as a known, deliberately-deferred issue.)

## Ideas / Features (unscoped)

- **[app] Delay time-series chart and worst-10 list on the summary page (PR 3 of 3, depends on the web-server PR).** The summary page (`frontend/src/pages/SummaryPage.tsx`, `frontend/src/api/stats.ts`) only shows the last 7 days, so there is no week-over-week or month-over-month view. Add a time-series chart (`frontend/` React+Vite; the Flutter `app/train_dashboard` is legacy and untouched). Granularity is chosen from the range (up to ~2 months: day; up to ~1 year: week; otherwise month) and the user can override it, capped at ~120 points. The current incomplete bucket is marked "in progress". A metric selector offers avg delay, % delayed and % cancelled, with counts in the tooltip, plus a severity-band stacked view. Service-type toggles and a Total series are included. Below the chart, a "worst 10" list for the selected range; clicking a bucket shows that period's top 10 instead. The app fetches `/api/stats/daily` (one row per day and service type, sums and counts) and does the day/ISO-week/month bucketing, the Total series, avg = delay_sum/delay_count and the percentages itself. The chart fetches once per range change and does not poll.
  - app: add `getDaily`/`getWorstTrips` to `api/stats.ts`; chart component with range/granularity/metric/service-type controls; worst-10 list wired to range and bucket click.
  - Acceptance: default view renders without error against DEV data; auto-granularity switches at the thresholds above and the override works; the in-progress bucket is visibly marked; toggling a service type or Total shows or hides its series; clicking a bucket updates the worst-10 list to that period.

## Decisions

- Pushing/committing always stays a human-confirmed step in the main session, never delegated to an autonomous agent.

## Fixed

- **[scraper] Midnight date-reconstruction + split-row bugs.** Both fixed together: `resolveClockTime` (in `utils.go`) now picks whichever of {yesterday, today, tomorrow} + CP's `"HH:MM"` clock string lands closest to `now`, instead of always assuming today's date. The row ID is now derived per-trip instead of once per run; when a trip's arrival is captured in the early morning (`isEarlyMorning`, hour < 6 — one-sided, since a late-night arrival can never predate midnight), the scraper looks up the still-open departure row for that train number (`findOpenTripID`) and reuses its ID instead of guessing, so departure and arrival merge into one row even when captured in different cron runs across midnight. Going-forward fix only — existing split/dropped rows in the DB were not backfilled.
- **[scraper] One failed station aborts the whole run.** `getAndStoreTrips` (`backend/scraper/service.go`) used to `return err` immediately when `cpClient.FetchTrips` failed for a station, skipping every subsequent station in `shared.Stations` for that cycle. Now it logs the error (`fmt.Printf`, same style as the `InsertStartingTrips`/`InsertEndingTrips` error handling just below) and `continue`s to the next station, so a single flaky CP API call no longer loses data for the rest of the run.
