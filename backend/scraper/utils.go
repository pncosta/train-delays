package main

import (
	"fmt"
	"time"
)

func Filter[T any](ss []T, test func(T) bool) (ret []T) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

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

// earlyMorningHours bounds how long after midnight an arrival can still plausibly
// belong to a trip that departed - and was recorded - the previous calendar day.
// An arrival captured late at night (e.g. 23:50) can never predate midnight, so this
// risk is one-sided: only early-morning arrivals need the extra open-row lookup.
// CP's longest routes run well under this many hours, so it's a safe bound without
// having to run the lookup on every single ending-trip insert.
const earlyMorningHours = 6

func isEarlyMorning(now time.Time) bool {
	return now.Hour() < earlyMorningHours
}
