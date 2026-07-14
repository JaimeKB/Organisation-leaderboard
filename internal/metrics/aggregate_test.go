package metrics

import (
	"testing"
	"time"

	"github.com/jaimekb/org-leaderboard/internal/githubapi"
)

func TestCommitCountMetric_ComputeFiltersToWindow(t *testing.T) {
	window := TimeWindow{
		Since: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Until: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
	}
	stats := []githubapi.ContributorStats{
		{
			Login: "alice",
			Weeks: []githubapi.WeeklyStat{
				{WeekStart: time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC), Commits: 100}, // outside window
				{WeekStart: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), Commits: 5},
				{WeekStart: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), Commits: 3},
			},
		},
		{
			Login: "bob",
			Weeks: []githubapi.WeeklyStat{
				{WeekStart: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), Commits: 0},
			},
		},
	}

	got := CommitCountMetric{}.Compute(stats, window)

	if got["alice"] != 8 {
		t.Errorf("alice commits = %v, want 8", got["alice"])
	}
	if _, ok := got["bob"]; ok {
		t.Errorf("bob should be omitted when total is 0, got %v", got["bob"])
	}
}

func TestAggregate_SumsAcrossRepos(t *testing.T) {
	perRepo := []RepoMetricResult{
		{RepoFullName: "org/repo1", MetricKey: "commits", Values: map[string]float64{"alice": 5, "bob": 2}},
		{RepoFullName: "org/repo2", MetricKey: "commits", Values: map[string]float64{"alice": 3, "carol": 10}},
	}

	result := Aggregate(perRepo, []Metric{CommitCountMetric{}})

	byLogin := map[string]ContributorResult{}
	for _, row := range result.Rows {
		byLogin[row.Login] = row
	}

	if byLogin["alice"].Values["commits"] != 8 {
		t.Errorf("alice total = %v, want 8", byLogin["alice"].Values["commits"])
	}
	if byLogin["bob"].Values["commits"] != 2 {
		t.Errorf("bob total = %v, want 2", byLogin["bob"].Values["commits"])
	}
	if byLogin["carol"].Values["commits"] != 10 {
		t.Errorf("carol total = %v, want 10", byLogin["carol"].Values["commits"])
	}
}

func TestLeaderboardResult_SortedByDescending(t *testing.T) {
	result := LeaderboardResult{
		Rows: []ContributorResult{
			{Login: "bob", Values: map[string]float64{"commits": 2}},
			{Login: "alice", Values: map[string]float64{"commits": 8}},
			{Login: "carol", Values: map[string]float64{"commits": 10}},
		},
	}

	sorted := result.SortedBy("commits")

	want := []string{"carol", "alice", "bob"}
	for i, w := range want {
		if sorted[i].Login != w {
			t.Errorf("sorted[%d] = %s, want %s", i, sorted[i].Login, w)
		}
	}
}
