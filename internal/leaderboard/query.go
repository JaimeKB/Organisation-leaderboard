// Package leaderboard aggregates per-repo commit data into a ranked list of
// contributors, driven by a single filterable query.
package leaderboard

import "time"

// Query is the single input to Compute. New filter dimensions (branch, file
// path, exclude-bots, etc.) should be added here as extra fields, with
// matching predicate logic in Compute — the API handler and filter UI are
// the only other places that need to know about a new field.
type Query struct {
	Org string

	// Repos restricts the leaderboard to these repo names (within Org).
	// Empty means every repo in the org.
	Repos []string

	// Users restricts the leaderboard to these GitHub logins.
	// Empty means every contributor.
	Users []string

	Since time.Time
	Until time.Time
}

// DefaultWindow is used when a query doesn't specify a date range.
const DefaultWindow = 90 * 24 * time.Hour

func (q Query) userSet() map[string]bool {
	if len(q.Users) == 0 {
		return nil
	}
	set := make(map[string]bool, len(q.Users))
	for _, u := range q.Users {
		set[u] = true
	}
	return set
}
