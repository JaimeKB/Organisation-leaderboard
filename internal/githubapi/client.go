// Package githubapi wraps the subset of the GitHub REST API this app needs:
// listing the authenticated user's orgs, the repos they can see within an
// org, org members, and commits for the leaderboard tally.
package githubapi

import (
	"context"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

type Client struct {
	gh *github.Client
}

func New(ctx context.Context, token string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, ts)
	return &Client{gh: github.NewClient(httpClient)}
}

// RateLimit returns the current core rate limit status, primarily so the UI
// can surface remaining quota to the user.
func (c *Client) RateLimit(ctx context.Context) (*github.RateLimits, error) {
	limits, _, err := c.gh.RateLimit.Get(ctx)
	return limits, err
}
