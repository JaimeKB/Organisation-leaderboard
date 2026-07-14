// Package metrics computes per-contributor leaderboard values from GitHub
// contributor stats. New metrics (lines changed, PR count, etc.) can be
// added by implementing Metric and registering it in registry.go, without
// changing the pipeline or leaderboard rendering code.
package metrics

import (
	"time"

	"github.com/jaimekb/org-leaderboard/internal/githubapi"
)

// TimeWindow bounds the period a Metric should consider, e.g. "last 3
// months". Until is exclusive.
type TimeWindow struct {
	Since time.Time
	Until time.Time
}

// Contains reports whether t falls within the window.
func (w TimeWindow) Contains(t time.Time) bool {
	return !t.Before(w.Since) && t.Before(w.Until)
}

// LastNMonths returns a TimeWindow covering the n months up to now.
func LastNMonths(n int) TimeWindow {
	now := time.Now().UTC()
	return TimeWindow{Since: now.AddDate(0, -n, 0), Until: now}
}

// Metric is a pluggable strategy for computing one leaderboard column from a
// single repo's contributor stats. Values for the same contributor across
// multiple repos are summed by Aggregate.
type Metric interface {
	// Key is a stable identifier used as a map key and in the ?sort= query param.
	Key() string
	// DisplayName is the human-readable column header.
	DisplayName() string
	// Compute returns login -> value for one repo's stats, restricted to window.
	Compute(stats []githubapi.ContributorStats, window TimeWindow) map[string]float64
}
