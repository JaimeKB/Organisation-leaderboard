package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jaimekb/org-leaderboard/internal/githubapi"
	"github.com/jaimekb/org-leaderboard/internal/metrics"
)

type fakeClient struct {
	stats map[string][]githubapi.ContributorStats
	errs  map[string]error

	active    int32
	maxActive int32
	delay     time.Duration
}

func (f *fakeClient) ListUserOrgs(ctx context.Context) ([]githubapi.Org, error) { return nil, nil }
func (f *fakeClient) ListOrgRepos(ctx context.Context, org string) ([]githubapi.Repo, error) {
	return nil, nil
}

func (f *fakeClient) ListContributorStats(ctx context.Context, owner, repo string) ([]githubapi.ContributorStats, error) {
	cur := atomic.AddInt32(&f.active, 1)
	defer atomic.AddInt32(&f.active, -1)
	for {
		max := atomic.LoadInt32(&f.maxActive)
		if cur <= max || atomic.CompareAndSwapInt32(&f.maxActive, max, cur) {
			break
		}
	}
	if f.delay > 0 {
		time.Sleep(f.delay)
	}

	key := owner + "/" + repo
	if err, ok := f.errs[key]; ok {
		return nil, err
	}
	return f.stats[key], nil
}

func statsFor(login string, commits int) githubapi.ContributorStats {
	return githubapi.ContributorStats{
		Login: login,
		Weeks: []githubapi.WeeklyStat{
			{WeekStart: time.Now().Add(-time.Hour), Commits: commits},
		},
	}
}

func TestRun_AggregatesAcrossRepos(t *testing.T) {
	client := &fakeClient{
		stats: map[string][]githubapi.ContributorStats{
			"org/repo1": {statsFor("alice", 5), statsFor("bob", 2)},
			"org/repo2": {statsFor("alice", 3), statsFor("carol", 10)},
		},
	}
	repos := []RepoRef{{Owner: "org", Name: "repo1"}, {Owner: "org", Name: "repo2"}}

	result, err := Run(context.Background(), client, repos, metrics.Enabled(), Options{
		Window: metrics.LastNMonths(3),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	byLogin := map[string]metrics.ContributorResult{}
	for _, row := range result.Leaderboard.Rows {
		byLogin[row.Login] = row
	}

	if byLogin["alice"].Values["commits"] != 8 {
		t.Errorf("alice = %v, want 8", byLogin["alice"].Values["commits"])
	}
	if byLogin["carol"].Values["commits"] != 10 {
		t.Errorf("carol = %v, want 10", byLogin["carol"].Values["commits"])
	}
	if len(result.RepoStatuses) != 2 {
		t.Errorf("len(RepoStatuses) = %d, want 2", len(result.RepoStatuses))
	}
	for _, s := range result.RepoStatuses {
		if s.Err != nil || s.Pending {
			t.Errorf("unexpected status for %v: %+v", s.Repo, s)
		}
	}
}

func TestRun_RecordsPendingAndErrorStatusesWithoutFailingRun(t *testing.T) {
	client := &fakeClient{
		stats: map[string][]githubapi.ContributorStats{
			"org/good": {statsFor("alice", 5)},
		},
		errs: map[string]error{
			"org/pending": githubapi.ErrStatsPending,
			"org/broken":  errors.New("boom"),
		},
	}
	repos := []RepoRef{
		{Owner: "org", Name: "good"},
		{Owner: "org", Name: "pending"},
		{Owner: "org", Name: "broken"},
	}

	result, err := Run(context.Background(), client, repos, metrics.Enabled(), Options{
		Window: metrics.LastNMonths(3),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(result.Leaderboard.Rows) != 1 || result.Leaderboard.Rows[0].Login != "alice" {
		t.Fatalf("Leaderboard.Rows = %+v, want only alice", result.Leaderboard.Rows)
	}

	var sawPending, sawErr bool
	for _, s := range result.RepoStatuses {
		switch s.Repo.Name {
		case "pending":
			sawPending = s.Pending
		case "broken":
			sawErr = s.Err != nil
		}
	}
	if !sawPending {
		t.Errorf("expected pending repo to be flagged Pending")
	}
	if !sawErr {
		t.Errorf("expected broken repo to be flagged with an Err")
	}
}

func TestRun_RespectsConcurrencyLimit(t *testing.T) {
	client := &fakeClient{
		stats: map[string][]githubapi.ContributorStats{},
		delay: 20 * time.Millisecond,
	}
	var repos []RepoRef
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("repo%d", i)
		repos = append(repos, RepoRef{Owner: "org", Name: name})
	}

	const limit = 3
	_, err := Run(context.Background(), client, repos, metrics.Enabled(), Options{
		Concurrency: limit,
		Window:      metrics.LastNMonths(3),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := atomic.LoadInt32(&client.maxActive); got > limit {
		t.Errorf("max concurrent calls = %d, want <= %d", got, limit)
	}
}
