package metrics

import "sort"

// RepoMetricResult is one metric's computed values for one repo, produced by
// the pipeline for each (repo, enabled metric) pair.
type RepoMetricResult struct {
	RepoFullName string
	MetricKey    string
	Values       map[string]float64 // login -> value
}

// ContributorResult is one contributor's totals across all selected repos,
// keyed by metric.
type ContributorResult struct {
	Login  string
	Values map[string]float64 // metricKey -> total
}

// LeaderboardResult is the full leaderboard: which metrics are shown, and
// each contributor's totals.
type LeaderboardResult struct {
	Metrics []Metric
	Rows    []ContributorResult
}

// Aggregate sums per-repo metric results into per-contributor totals across
// all repos. The returned Rows are in unspecified order; use SortedBy to
// order them for display.
func Aggregate(perRepo []RepoMetricResult, enabled []Metric) LeaderboardResult {
	totals := make(map[string]map[string]float64)
	for _, r := range perRepo {
		for login, val := range r.Values {
			if totals[login] == nil {
				totals[login] = make(map[string]float64)
			}
			totals[login][r.MetricKey] += val
		}
	}

	rows := make([]ContributorResult, 0, len(totals))
	for login, values := range totals {
		rows = append(rows, ContributorResult{Login: login, Values: values})
	}

	return LeaderboardResult{Metrics: enabled, Rows: rows}
}

// SortedBy returns Rows ordered descending by the given metric key. Rows
// missing that metric sort last. The receiver's Rows are left untouched.
func (r LeaderboardResult) SortedBy(metricKey string) []ContributorResult {
	rows := make([]ContributorResult, len(r.Rows))
	copy(rows, r.Rows)
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Values[metricKey] > rows[j].Values[metricKey]
	})
	return rows
}
