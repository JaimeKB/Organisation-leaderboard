// Package githubapi is the sole seam between this application's business
// logic and the GitHub API (via google/go-github). Business logic depends
// only on the Client interface, never on *github.Client directly, so
// handlers/pipeline/metrics can be unit tested against a fake.
package githubapi

import (
	"context"
	"net/url"
	"time"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

// Org is a GitHub organization the authenticated user belongs to.
type Org struct {
	Login string
	ID    int64
}

// Repo is a GitHub repository within an org.
type Repo struct {
	Owner    string
	Name     string
	FullName string
	Private  bool
}

// WeeklyStat is one week of a contributor's activity on a repo.
type WeeklyStat struct {
	WeekStart time.Time
	Additions int
	Deletions int
	Commits   int
}

// ContributorStats is one contributor's weekly activity on a repo, as
// returned by the GitHub contributor-stats endpoint (up to 52 weeks).
type ContributorStats struct {
	Login string
	Weeks []WeeklyStat
}

// Client is the GitHub API surface this application needs. The real
// implementation wraps google/go-github; tests use a fake implementing the
// same interface.
type Client interface {
	// ListUserOrgs lists the organizations the authenticated user belongs to.
	ListUserOrgs(ctx context.Context) ([]Org, error)
	// ListOrgRepos lists all repositories within org.
	ListOrgRepos(ctx context.Context, org string) ([]Repo, error)
	// ListContributorStats returns per-contributor weekly stats for a repo,
	// retrying while GitHub computes them asynchronously (202 responses).
	ListContributorStats(ctx context.Context, owner, repo string) ([]ContributorStats, error)
}

type client struct {
	gh *github.Client
}

// NewClient builds a Client authenticated with a GitHub Personal Access
// Token, talking to the real GitHub API.
func NewClient(pat string) Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: pat})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return &client{gh: github.NewClient(httpClient)}
}

// NewClientWithBaseURL is NewClient but pointed at an arbitrary base URL,
// used by tests to target an httptest.Server instead of the real API.
func NewClientWithBaseURL(pat, baseURL string) (Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: pat})
	httpClient := oauth2.NewClient(context.Background(), ts)
	gh := github.NewClient(httpClient)

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	gh.BaseURL = parsed
	return &client{gh: gh}, nil
}
