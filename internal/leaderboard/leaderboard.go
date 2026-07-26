package leaderboard

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/go-github/v66/github"
	"golang.org/x/sync/errgroup"
)

// maxConcurrentRepoFetches bounds how many repos are fetched in parallel, so
// a large org doesn't blow through the GitHub rate limit or open too many
// connections at once.
const maxConcurrentRepoFetches = 5

// Fetcher is the GitHub access Compute needs. githubapi.Client satisfies it;
// tests supply a fake so aggregation logic can be verified without network
// access.
type Fetcher interface {
	ListReposByOrg(ctx context.Context, org string) ([]*github.Repository, error)
	ListCommits(ctx context.Context, owner, repo string, since, until time.Time) ([]*github.RepositoryCommit, error)
}

// Contributor is one row of the leaderboard.
type Contributor struct {
	Login       string         `json:"login"`
	Name        string         `json:"name"`
	AvatarURL   string         `json:"avatarUrl"`
	CommitCount int            `json:"commitCount"`
	Repos       map[string]int `json:"repos"` // per-repo breakdown, for future drill-down
}

// Compute fetches commits for the repos/window in q (bounded concurrency
// across repos), tallies them per author, applies the user filter, and
// returns contributors sorted by commit count descending.
func Compute(ctx context.Context, f Fetcher, q Query) ([]Contributor, error) {
	since, until := resolveWindow(q)

	repoNames, err := resolveRepoNames(ctx, f, q)
	if err != nil {
		return nil, err
	}

	tally, err := fetchAndTally(ctx, f, q.Org, repoNames, since, until, q.userSet())
	if err != nil {
		return nil, err
	}

	return sortedContributors(tally), nil
}

func resolveWindow(q Query) (since, until time.Time) {
	until = q.Until
	if until.IsZero() {
		until = time.Now()
	}
	since = q.Since
	if since.IsZero() {
		since = until.Add(-DefaultWindow)
	}
	return since, until
}

func resolveRepoNames(ctx context.Context, f Fetcher, q Query) ([]string, error) {
	if len(q.Repos) > 0 {
		return q.Repos, nil
	}
	repos, err := f.ListReposByOrg(ctx, q.Org)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(repos))
	for _, r := range repos {
		names = append(names, r.GetName())
	}
	return names, nil
}

// fetchAndTally fetches commits for each repo concurrently (bounded by
// maxConcurrentRepoFetches) and merges them into a single tally, guarded by
// a mutex since goroutines write to it concurrently.
func fetchAndTally(ctx context.Context, f Fetcher, org string, repoNames []string, since, until time.Time, userFilter map[string]bool) (map[string]*Contributor, error) {
	var mu sync.Mutex
	tally := make(map[string]*Contributor)

	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, maxConcurrentRepoFetches)

	for _, repoName := range repoNames {
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			commits, err := f.ListCommits(gctx, org, repoName, since, until)
			if err != nil {
				return err
			}

			mu.Lock()
			defer mu.Unlock()
			addCommits(tally, commits, repoName, userFilter)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return tally, nil
}

func addCommits(tally map[string]*Contributor, commits []*github.RepositoryCommit, repoName string, userFilter map[string]bool) {
	for _, c := range commits {
		login, name, avatar := authorOf(c)
		if userFilter != nil && !userFilter[login] {
			continue
		}
		entry, ok := tally[login]
		if !ok {
			entry = &Contributor{Login: login, Name: name, AvatarURL: avatar, Repos: map[string]int{}}
			tally[login] = entry
		}
		entry.CommitCount++
		entry.Repos[repoName]++
	}
}

func sortedContributors(tally map[string]*Contributor) []Contributor {
	result := make([]Contributor, 0, len(tally))
	for _, c := range tally {
		result = append(result, *c)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CommitCount != result[j].CommitCount {
			return result[i].CommitCount > result[j].CommitCount
		}
		return result[i].Login < result[j].Login
	})
	return result
}

// authorOf resolves the GitHub login for a commit where possible, falling
// back to the raw git author name/email for commits not linked to a GitHub
// account (e.g. made with an email not associated with any account).
func authorOf(c *github.RepositoryCommit) (login, name, avatar string) {
	if a := c.GetAuthor(); a != nil && a.GetLogin() != "" {
		return a.GetLogin(), a.GetLogin(), a.GetAvatarURL()
	}
	if commit := c.GetCommit(); commit != nil && commit.GetAuthor() != nil {
		ca := commit.GetAuthor()
		n := ca.GetName()
		if n == "" {
			n = ca.GetEmail()
		}
		if n == "" {
			n = "Unknown"
		}
		return "email:" + ca.GetEmail(), n, ""
	}
	return "unknown", "Unknown", ""
}
