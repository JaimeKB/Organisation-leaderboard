package githubapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/go-github/v66/github"
)

// statsRetryDelays are the backoff delays between retries while GitHub
// computes contributor stats asynchronously (~30-45s total budget).
var statsRetryDelays = []time.Duration{
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
}

// ErrStatsPending is returned by ListContributorStats when GitHub is still
// computing stats for a repo after exhausting the retry budget. Callers
// should treat this as "try again later" rather than a hard failure.
var ErrStatsPending = errors.New("contributor stats still being computed by GitHub")

// ListContributorStats returns per-contributor weekly commit/addition/
// deletion stats for a repo. GitHub computes these asynchronously for repos
// that haven't been queried recently; while pending, the API returns 202
// which go-github surfaces as *github.AcceptedError. This retries with
// backoff and returns ErrStatsPending if still not ready after the budget.
func (c *client) ListContributorStats(ctx context.Context, owner, repo string) ([]ContributorStats, error) {
	for attempt := 0; ; attempt++ {
		stats, _, err := c.gh.Repositories.ListContributorsStats(ctx, owner, repo)
		if err == nil {
			return convertContributorStats(stats), nil
		}

		var accepted *github.AcceptedError
		if !errors.As(err, &accepted) {
			return nil, fmt.Errorf("fetching contributor stats for %s/%s: %w", owner, repo, err)
		}

		if attempt >= len(statsRetryDelays) {
			return nil, ErrStatsPending
		}

		select {
		case <-time.After(statsRetryDelays[attempt]):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func convertContributorStats(in []*github.ContributorStats) []ContributorStats {
	out := make([]ContributorStats, 0, len(in))
	for _, s := range in {
		if s == nil || s.Author == nil {
			continue
		}
		weeks := make([]WeeklyStat, 0, len(s.Weeks))
		for _, w := range s.Weeks {
			if w == nil {
				continue
			}
			weekStart := time.Time{}
			if w.Week != nil {
				weekStart = w.Week.Time
			}
			weeks = append(weeks, WeeklyStat{
				WeekStart: weekStart,
				Additions: w.GetAdditions(),
				Deletions: w.GetDeletions(),
				Commits:   w.GetCommits(),
			})
		}
		out = append(out, ContributorStats{
			Login: s.Author.GetLogin(),
			Weeks: weeks,
		})
	}
	return out
}
