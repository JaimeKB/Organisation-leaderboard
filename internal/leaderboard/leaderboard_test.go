package leaderboard

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v66/github"
)

type fakeFetcher struct {
	repos       []*github.Repository
	commitsByRepo map[string][]*github.RepositoryCommit
}

func (f *fakeFetcher) ListReposByOrg(ctx context.Context, org string) ([]*github.Repository, error) {
	return f.repos, nil
}

func (f *fakeFetcher) ListCommits(ctx context.Context, owner, repo string, since, until time.Time) ([]*github.RepositoryCommit, error) {
	return f.commitsByRepo[repo], nil
}

func ghAuthorCommit(login string) *github.RepositoryCommit {
	return &github.RepositoryCommit{
		Author: &github.User{Login: github.String(login)},
		Commit: &github.Commit{Author: &github.CommitAuthor{Name: github.String(login)}},
	}
}

func unlinkedCommit(name, email string) *github.RepositoryCommit {
	return &github.RepositoryCommit{
		Commit: &github.Commit{Author: &github.CommitAuthor{Name: github.String(name), Email: github.String(email)}},
	}
}

func TestCompute_TalliesAndSorts(t *testing.T) {
	f := &fakeFetcher{
		repos: []*github.Repository{{Name: github.String("repo-a")}, {Name: github.String("repo-b")}},
		commitsByRepo: map[string][]*github.RepositoryCommit{
			"repo-a": {ghAuthorCommit("alice"), ghAuthorCommit("alice"), ghAuthorCommit("bob")},
			"repo-b": {ghAuthorCommit("bob"), ghAuthorCommit("bob")},
		},
	}

	got, err := Compute(context.Background(), f, Query{Org: "acme"})
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Login != "bob" || got[0].CommitCount != 3 {
		t.Errorf("rank 0 = %+v, want bob:3", got[0])
	}
	if got[1].Login != "alice" || got[1].CommitCount != 2 {
		t.Errorf("rank 1 = %+v, want alice:2", got[1])
	}
	if got[0].Repos["repo-a"] != 1 || got[0].Repos["repo-b"] != 2 {
		t.Errorf("bob per-repo breakdown = %+v", got[0].Repos)
	}
}

func TestCompute_FiltersByRepo(t *testing.T) {
	f := &fakeFetcher{
		commitsByRepo: map[string][]*github.RepositoryCommit{
			"repo-a": {ghAuthorCommit("alice")},
			"repo-b": {ghAuthorCommit("bob")},
		},
	}

	got, err := Compute(context.Background(), f, Query{Org: "acme", Repos: []string{"repo-a"}})
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if len(got) != 1 || got[0].Login != "alice" {
		t.Fatalf("got = %+v, want only alice", got)
	}
}

func TestCompute_FiltersByUser(t *testing.T) {
	f := &fakeFetcher{
		repos: []*github.Repository{{Name: github.String("repo-a")}},
		commitsByRepo: map[string][]*github.RepositoryCommit{
			"repo-a": {ghAuthorCommit("alice"), ghAuthorCommit("bob")},
		},
	}

	got, err := Compute(context.Background(), f, Query{Org: "acme", Users: []string{"bob"}})
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if len(got) != 1 || got[0].Login != "bob" {
		t.Fatalf("got = %+v, want only bob", got)
	}
}

func TestCompute_FallsBackToGitAuthorForUnlinkedCommits(t *testing.T) {
	f := &fakeFetcher{
		repos: []*github.Repository{{Name: github.String("repo-a")}},
		commitsByRepo: map[string][]*github.RepositoryCommit{
			"repo-a": {unlinkedCommit("Jane Doe", "jane@example.com")},
		},
	}

	got, err := Compute(context.Background(), f, Query{Org: "acme"})
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Login != "email:jane@example.com" || got[0].Name != "Jane Doe" {
		t.Errorf("got = %+v", got[0])
	}
}

func TestCompute_DefaultsDateWindow(t *testing.T) {
	f := &fakeFetcher{repos: []*github.Repository{{Name: github.String("repo-a")}}, commitsByRepo: map[string][]*github.RepositoryCommit{}}

	got, err := Compute(context.Background(), f, Query{Org: "acme"})
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v, want empty", got)
	}
}
