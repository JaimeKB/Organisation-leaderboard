package githubapi

import (
	"context"
	"fmt"

	"github.com/google/go-github/v66/github"
)

// ListReposByOrg returns every repository within org that the authenticated
// user's token can see. GitHub itself scopes the response to what the token
// has access to, so no extra filtering is needed here.
func (c *Client) ListReposByOrg(ctx context.Context, org string) ([]*github.Repository, error) {
	var all []*github.Repository
	opts := &github.RepositoryListByOrgOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		repos, resp, err := c.gh.Repositories.ListByOrg(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("list repos for org %s: %w", org, err)
		}
		all = append(all, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}
