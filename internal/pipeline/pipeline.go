// Package pipeline fetches contributor stats for a set of repos with bounded
// concurrency, computes the enabled metrics for each, and aggregates the
// results into a leaderboard. Callers should wrap ctx with a deadline (e.g.
// 3-5 minutes) so a stuck GitHub call can't hang the wizard forever.
package pipeline

import (
	"context"
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/jaimekb/org-leaderboard/internal/githubapi"
	"github.com/jaimekb/org-leaderboard/internal/metrics"
)

// RepoRef identifies a repo to fetch stats for.
type RepoRef struct {
	Owner string
	Name  string
}

// FullName returns "owner/name".
func (r RepoRef) FullName() string { return r.Owner + "/" + r.Name }

// Options configures a pipeline run.
type Options struct {
	// Concurrency bounds simultaneous ListContributorStats calls. Defaults
	// to 5 if <= 0.
	Concurrency int
	// Window is the time range metrics are computed over (e.g. last 3 months).
	Window metrics.TimeWindow
}

// RepoStatus reports how fetching one repo's stats went, so the UI can
// distinguish "no data" from "still computing" from "failed".
type RepoStatus struct {
	Repo    RepoRef
	Pending bool // GitHub hadn't finished computing stats within the retry budget
	Err     error
}

// Result is a completed pipeline run: the aggregated leaderboard plus
// per-repo fetch status so partial failures are visible rather than silent.
type Result struct {
	Leaderboard  metrics.LeaderboardResult
	RepoStatuses []RepoStatus
}

// Run fetches contributor stats for repos concurrently (bounded by
// opts.Concurrency), computes every metric in enabled for each repo, and
// aggregates into a leaderboard. A single repo's failure (including stats
// still being computed by GitHub) is recorded in RepoStatuses rather than
// failing the whole run.
func Run(ctx context.Context, client githubapi.Client, repos []RepoRef, enabled []metrics.Metric, opts Options) (Result, error) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	var (
		mu       sync.Mutex
		results  []metrics.RepoMetricResult
		statuses []RepoStatus
	)

	for _, repo := range repos {
		g.Go(func() error {
			stats, err := client.ListContributorStats(gctx, repo.Owner, repo.Name)
			if err != nil {
				status := RepoStatus{Repo: repo}
				if errors.Is(err, githubapi.ErrStatsPending) {
					status.Pending = true
				} else {
					status.Err = err
				}
				mu.Lock()
				statuses = append(statuses, status)
				mu.Unlock()
				return nil
			}

			repoResults := make([]metrics.RepoMetricResult, 0, len(enabled))
			for _, m := range enabled {
				repoResults = append(repoResults, metrics.RepoMetricResult{
					RepoFullName: repo.FullName(),
					MetricKey:    m.Key(),
					Values:       m.Compute(stats, opts.Window),
				})
			}

			mu.Lock()
			results = append(results, repoResults...)
			statuses = append(statuses, RepoStatus{Repo: repo})
			mu.Unlock()
			return nil
		})
	}

	// g.Wait only errors if a worker returns a non-nil error, which none do
	// above (per-repo failures are recorded in statuses instead) — it can
	// still surface ctx cancellation/timeout from the caller.
	if err := g.Wait(); err != nil {
		return Result{}, err
	}

	return Result{
		Leaderboard:  metrics.Aggregate(results, enabled),
		RepoStatuses: statuses,
	}, nil
}
