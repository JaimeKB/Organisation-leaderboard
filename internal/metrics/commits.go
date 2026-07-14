package metrics

import "github.com/jaimekb/org-leaderboard/internal/githubapi"

// CommitCountMetric totals commit counts per contributor within the window.
type CommitCountMetric struct{}

func (CommitCountMetric) Key() string         { return "commits" }
func (CommitCountMetric) DisplayName() string { return "Commits" }

func (CommitCountMetric) Compute(stats []githubapi.ContributorStats, window TimeWindow) map[string]float64 {
	values := make(map[string]float64)
	for _, s := range stats {
		var total int
		for _, w := range s.Weeks {
			if window.Contains(w.WeekStart) {
				total += w.Commits
			}
		}
		if total > 0 {
			values[s.Login] = float64(total)
		}
	}
	return values
}
