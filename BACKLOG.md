# Backlog

Dumping ground for ideas, known bugs, and improvements for pt-train-delays. Not a sprint board — just so nothing good gets lost. Add freely; prune when stale.

## Bugs

- **[scraper] DB connection opened/closed per insert batch.** `InsertEndingTrips`/`InsertStartingTrips` in `backend/scraper/repository.go` call `sql.Open` + `defer Close()` every time, ~2x per station per run (~100/run). Same class of issue as the HTTP client reuse fix already shipped for the CP API client (`a2bf6ae`) — just not applied to the DB side.
- **[scraper] Unchecked error from `http.NewRequestWithContext`** in `cp_client.go` — if it ever errors, the following `req.Header.Set` calls would panic on a nil `req`. Low likelihood, cheap to fix.

## Infra / Cost

- **Turso is billed per row read**, and the planned app usage pattern (every app-open triggers many reads) would get expensive fast. Need an aggregation/caching layer in the web-server before real users show up. (Flagged by Pedro as a known, deliberately-deferred issue.)

## Ideas / Features (unscoped)

- **[web-server] Time-series and worst-trips endpoints over a lazy per-day cache (PR 2 of 3, depends on the rollup PR).**
  - **Endpoints:**
    - `GET /api/stats/timeseries?granularity=day|week|month&from&to` returns, per period per `service_type` plus a `TOTAL` entry (sum of the per-type counts): trips, cancelled, on_time, delayed, avg_delay (`delay_sum / delay_count`), the four band counts and max delay. Weeks are ISO (Mon-Sun) and labelled by their Monday date; months are calendar months. Points are capped at ~120.
    - `GET /api/stats/worst-trips?from&to&service_type` returns up to 10 trips (service_type optional, otherwise all types), ordered by `delay_minutes DESC, train_number`.
  - **Caching:** both endpoints are served from an in-memory per-day cache of `daily_stats` and `daily_worst_trips` rows. The cache is lazy, so nothing is read at startup.
    - A request reads only the days of its range that aren't cached yet, in one `day`-range query, and caches them, including days with no data so they aren't re-asked.
    - Day, week and month buckets and a period's top 10 are built in Go from the cached days.
    - Past days never change after the 04:00 rollup, so cached days are never re-fetched. Yesterday appears on the first request after the rollup.
    - A day is cached as final only if it is before yesterday, or it is yesterday and its rollup rows exist; an empty yesterday/today is never cached as final (re-checked on later requests).
    - Concurrent first requests for the same days fetch once (mutex/`singleflight`).
    - No traffic means no reads. A backfill re-run needs a web-server restart or redeploy to clear the cache.
  - **`/summary`:** switch `GetSummary` in `backend/web-server/repository.go` to the same per-type thresholds. It currently uses `delay_minutes < 3` / `>= 3` for all types, but it should be `U` > 3 and all others > 5.
  - web-server: per-day cache for both tables with lazy missing-day fill; Go bucketing (day/ISO week/month, TOTAL) and top-10 merge; new handlers and `DBClient` range reads; validate `granularity`/`from`/`to` and cap points; update `GetSummary` thresholds.
  - Acceptance: no DB reads at startup. A repeated request makes zero DB reads. A request overlapping cached days reads only the uncached days. Week and month totals equal the sum of the underlying days for every metric. The worst-trips top 10 for a range equals the top 10 computed from raw `trips` over the same days. Yesterday shows up after the 04:00 rollup without a restart. An invalid `granularity` or range returns 400. `/summary` classifies a 4-min `U` trip as delayed and a 4-min `IC` trip as on time.

- **[app] Delay time-series chart and worst-10 list on the summary page (PR 3 of 3, depends on the web-server PR).** The summary page (`frontend/src/pages/SummaryPage.tsx`, `frontend/src/api/stats.ts`) only shows the last 7 days, so there is no week-over-week or month-over-month view. Add a time-series chart (`frontend/` React+Vite; the Flutter `app/train_dashboard` is legacy and untouched). Granularity is chosen from the range (up to ~2 months: day; up to ~1 year: week; otherwise month) and the user can override it, capped at ~120 points. The current incomplete bucket is marked "in progress". A metric selector offers avg delay, % delayed and % cancelled, with counts in the tooltip, plus a severity-band stacked view. Service-type toggles and a Total series are included. Below the chart, a "worst 10" list for the selected range; clicking a bucket shows that period's top 10 instead. The chart fetches once per range/granularity change and does not poll.
  - app: add `getTimeseries`/`getWorstTrips` to `api/stats.ts`; chart component with range/granularity/metric/service-type controls; worst-10 list wired to range and bucket click.
  - Acceptance: default view renders without error against DEV data; auto-granularity switches at the thresholds above and the override works; the in-progress bucket is visibly marked; toggling a service type or Total shows or hides its series; clicking a bucket updates the worst-10 list to that period.

## Decisions

- Pushing/committing always stays a human-confirmed step in the main session, never delegated to an autonomous agent.

## Fixed

- **[scraper] Midnight date-reconstruction + split-row bugs.** Both fixed together: `resolveClockTime` (in `utils.go`) now picks whichever of {yesterday, today, tomorrow} + CP's `"HH:MM"` clock string lands closest to `now`, instead of always assuming today's date. The row ID is now derived per-trip instead of once per run; when a trip's arrival is captured in the early morning (`isEarlyMorning`, hour < 6 — one-sided, since a late-night arrival can never predate midnight), the scraper looks up the still-open departure row for that train number (`findOpenTripID`) and reuses its ID instead of guessing, so departure and arrival merge into one row even when captured in different cron runs across midnight. Going-forward fix only — existing split/dropped rows in the DB were not backfilled.
- **[scraper] One failed station aborts the whole run.** `getAndStoreTrips` (`backend/scraper/service.go`) used to `return err` immediately when `cpClient.FetchTrips` failed for a station, skipping every subsequent station in `shared.Stations` for that cycle. Now it logs the error (`fmt.Printf`, same style as the `InsertStartingTrips`/`InsertEndingTrips` error handling just below) and `continue`s to the next station, so a single flaky CP API call no longer loses data for the rest of the run.
