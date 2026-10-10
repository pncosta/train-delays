# Backlog

Dumping ground for ideas, known bugs, and improvements for pt-train-delays. Not a sprint board — just so nothing good gets lost. Add freely; prune when stale.

## Bugs

- **[scraper] DB connection opened/closed per insert batch.** `InsertCompletedTrips`/`FindCapturedIDs` in `backend/scraper/repository.go` call `sql.Open` + `defer Close()` every time - now just ~2 connections/run (down from ~100/run under the old per-station scraper), since the per-train poller batches into one write pass and one precheck query per run. Same class of issue as the HTTP client reuse fix already shipped for the CP API client (`a2bf6ae`) — just not applied to the DB side.
- **[scraper] Unchecked error from `http.NewRequestWithContext`** in `cp_client.go` — if it ever errors, the following `req.Header.Set` calls would panic on a nil `req`. Low likelihood, cheap to fix.

## Infra / Cost

- **Turso is billed per row read**, and the planned app usage pattern (every app-open triggers many reads) would get expensive fast. Need an aggregation/caching layer in the web-server before real users show up. (Flagged by Pedro as a known, deliberately-deferred issue.)

## Ideas / Features (unscoped)

- **[scraper] Blocked: bulk realtime endpoint (`POST realtime-api/trains/details`) would make the per-train-polling item above unnecessary — needs new credentials we don't have.** Found by reading `joaodcp/cp-rt-worker` (the backend behind https://comboios.live, an unrelated third-party CP dashboard): CP has a third endpoint, `POST https://api-gateway.cp.pt/cp/services/realtime-api/trains/details`, that takes a JSON array of train numbers in the body and returns **all of them in one call** — a dict keyed by train number with live `status` (including an explicit `"CANCELLED"` value, unlike anything we've observed on `travel-api`), `delay`, `speed`, `occupancy`, `lastStation`, and a `stops` map keyed by station code with `arrival`/`departure`/`arrDelay`/`depDelay`. One POST per cron tick would replace the entire paced/sequential per-train design above and remove the rate-limit problem outright. Blocker: this endpoint needs a separate credential set (`REALTIME_API_KEY`/`REALTIME_CONNECT_ID`/`REALTIME_CONNECT_SECRET` in the worker's code) distinct from our current `travel-api` key, and there's no known official CP developer portal to request it from — the worker's own repo/commits/author replies have no hint of how those credentials were obtained. Likely path: capture network traffic from CP's official mobile app while using its live train-tracking/map feature specifically (as opposed to the schedule-lookup screen, which is what backs our current `travel-api` key). Parked until credentials exist; revisit as a replacement for the item above if/when they do.

## Decisions

- Pushing/committing always stays a human-confirmed step in the main session, never delegated to an autonomous agent.

## Fixed

- **[scraper] Midnight date-reconstruction + split-row bugs.** Both fixed together: `resolveClockTime` (in `utils.go`) now picks whichever of {yesterday, today, tomorrow} + CP's `"HH:MM"` clock string lands closest to `now`, instead of always assuming today's date. The row ID is now derived per-trip instead of once per run; when a trip's arrival is captured in the early morning (`isEarlyMorning`, hour < 6 — one-sided, since a late-night arrival can never predate midnight), the scraper looks up the still-open departure row for that train number (`findOpenTripID`) and reuses its ID instead of guessing, so departure and arrival merge into one row even when captured in different cron runs across midnight. Going-forward fix only — existing split/dropped rows in the DB were not backfilled.
- **[scraper] One failed station aborts the whole run.** `getAndStoreTrips` (`backend/scraper/service.go`) used to `return err` immediately when `cpClient.FetchTrips` failed for a station, skipping every subsequent station in `shared.Stations` for that cycle. Now it logs the error (`fmt.Printf`, same style as the `InsertStartingTrips`/`InsertEndingTrips` error handling just below) and `continue`s to the next station, so a single flaky CP API call no longer loses data for the rest of the run.
