package metrics

import (
	"math"
	"sort"
	"time"
)

// CalculateHours estimates the work hours based on a slice of commit timestamps.
// The algorithm sorts timestamps and accumulates the diff if it's less than
// or equal to the session window. It assumes each standalone session start
// contributes a set minimum time (e.g. half the session window) to prevent
// 0 hour estimates for isolated commits.
func CalculateHours(timestamps []int64, sessionWindowMinutes int) int {
	if len(timestamps) == 0 {
		return 0
	}

	// Clone the slice to avoid mutating the caller's data
	ts := make([]int64, len(timestamps))
	copy(ts, timestamps)

	sort.SliceStable(ts, func(i, j int) bool {
		return ts[i] < ts[j]
	})

	var totalSeconds int64
	maxGapSecs := int64(sessionWindowMinutes * 60)
	minSessionTimeSecs := maxGapSecs / 2

	// Initially we assume the first commit took minSessionTimeSecs to write.
	totalSeconds += minSessionTimeSecs

	for i := 1; i < len(ts); i++ {
		diff := ts[i] - ts[i-1]
		if diff <= maxGapSecs {
			totalSeconds += diff
		} else {
			// New session
			totalSeconds += minSessionTimeSecs
		}
	}

	hours := int(math.Round(float64(totalSeconds) / 3600.0))
	return hours
}

// CalculateMonths counts unique active months from commit timestamps.
func CalculateMonths(timestamps []int64) int {
	uniqueMonths := make(map[string]bool)
	for _, ts := range timestamps {
		t := time.Unix(ts, 0).UTC()
		monthStr := t.Format("2006-01")
		uniqueMonths[monthStr] = true
	}
	return len(uniqueMonths)
}
