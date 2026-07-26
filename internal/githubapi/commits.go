package githubapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v66/github"
)

// ListCommits returns every commit on repo's default branch between since
// and until (either may be zero to leave that bound open).
func (c *Client) ListCommits(ctx context.Context, owner, repo string, since, until time.Time) ([]*github.RepositoryCommit, error) {
	var all []*github.RepositoryCommit
	opts := &github.CommitsListOptions{
		Since:       since,
		Until:       until,
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		commits, resp, err := c.gh.Repositories.ListCommits(ctx, owner, repo, opts)
		if err != nil {
			// An empty repository (no commits yet) returns 409; treat it as
			// zero commits rather than failing the whole leaderboard fetch.
			if resp != nil && resp.StatusCode == http.StatusConflict {
				return nil, nil
			}
			return nil, fmt.Errorf("list commits for %s/%s: %w", owner, repo, err)
		}
		all = append(all, commits...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}
