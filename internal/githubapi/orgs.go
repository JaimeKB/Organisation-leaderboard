package githubapi

import (
	"context"
	"fmt"

	"github.com/google/go-github/v66/github"
)

// ListUserOrgs lists all organizations the authenticated user belongs to,
// paginating until exhausted.
func (c *client) ListUserOrgs(ctx context.Context) ([]Org, error) {
	opts := &github.ListOptions{PerPage: 100}

	var orgs []Org
	for {
		page, resp, err := c.gh.Organizations.List(ctx, "", opts)
		if err != nil {
			return nil, fmt.Errorf("listing user orgs: %w", err)
		}
		for _, o := range page {
			orgs = append(orgs, Org{
				Login: o.GetLogin(),
				ID:    o.GetID(),
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return orgs, nil
}
