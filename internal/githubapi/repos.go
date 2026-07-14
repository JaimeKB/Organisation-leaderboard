package githubapi

import (
	"context"
	"fmt"

	"github.com/google/go-github/v66/github"
)

// ListOrgRepos lists all repositories within org, paginating until
// exhausted.
func (c *client) ListOrgRepos(ctx context.Context, org string) ([]Repo, error) {
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var repos []Repo
	for {
		page, resp, err := c.gh.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("listing repos for org %s: %w", org, err)
		}
		for _, r := range page {
			repos = append(repos, Repo{
				Owner:    org,
				Name:     r.GetName(),
				FullName: r.GetFullName(),
				Private:  r.GetPrivate(),
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return repos, nil
}
