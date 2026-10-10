package main

import (
	"fmt"
	"time"
)

// resolveClockTime turns a bare "HH:MM" clock string from the CP API into a full
// timestamp, by picking whichever of {yesterday, today, tomorrow} + clock lands
// closest to now. CP never tells us the calendar date of an event, only the time of
// day, so near midnight "23:50" or "00:05" can belong to either side of the day
// boundary - anchoring to the candidate closest to now resolves that ambiguity
// because events are always captured within ~45 minutes of actually happening.
func resolveClockTime(clock string, now time.Time) (time.Time, error) {
	var best time.Time
	var bestDiff time.Duration
	found := false

	for _, dayOffset := range []int{0, -1, 1} {
		day := now.AddDate(0, 0, dayOffset).Format("2006-01-02")
		candidate, err := time.ParseInLocation("2006-01-02 15:04", day+" "+clock, now.Location())
		if err != nil {
			continue
		}

		diff := candidate.Sub(now)
		if diff < 0 {
			diff = -diff
		}
		if !found || diff < bestDiff {
			best, bestDiff, found = candidate, diff, true
		}
	}

	if !found {
		return time.Time{}, fmt.Errorf("could not parse clock time %q", clock)
	}
	return best, nil
}
