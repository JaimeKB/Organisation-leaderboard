package githubapi

import (
	"context"
	"fmt"

	"github.com/google/go-github/v66/github"
)

// ListOrgs returns every organization the authenticated user belongs to.
func (c *Client) ListOrgs(ctx context.Context) ([]*github.Organization, error) {
	var all []*github.Organization
	opts := &github.ListOptions{PerPage: 100}
	for {
		orgs, resp, err := c.gh.Organizations.List(ctx, "", opts)
		if err != nil {
			return nil, fmt.Errorf("list orgs: %w", err)
		}
		all = append(all, orgs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

// ListOrgMembers returns the members of an org visible to the authenticated
// user (public members only, unless the user is an org owner).
func (c *Client) ListOrgMembers(ctx context.Context, org string) ([]*github.User, error) {
	var all []*github.User
	opts := &github.ListMembersOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		members, resp, err := c.gh.Organizations.ListMembers(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("list members of %s: %w", org, err)
		}
		all = append(all, members...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}
