package githubapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.Handler) Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := NewClientWithBaseURL("test-pat", srv.URL+"/")
	if err != nil {
		t.Fatalf("NewClientWithBaseURL() error = %v", err)
	}
	return c
}

func TestListUserOrgs_Paginates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `[{"login":"org-two","id":2}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s?page=2>; rel="next"`, r.URL.Path))
		fmt.Fprint(w, `[{"login":"org-one","id":1}]`)
	})

	c := newTestClient(t, mux)

	orgs, err := c.ListUserOrgs(context.Background())
	if err != nil {
		t.Fatalf("ListUserOrgs() error = %v", err)
	}
	if len(orgs) != 2 || orgs[0].Login != "org-one" || orgs[1].Login != "org-two" {
		t.Fatalf("ListUserOrgs() = %+v, want [org-one org-two]", orgs)
	}
}

func TestListOrgRepos_Paginates(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/orgs/my-org/repos", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `[{"name":"repo-two","full_name":"my-org/repo-two","private":true}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s?page=2>; rel="next"`, r.URL.Path))
		fmt.Fprint(w, `[{"name":"repo-one","full_name":"my-org/repo-one","private":false}]`)
	})

	c := newTestClient(t, mux)

	repos, err := c.ListOrgRepos(context.Background(), "my-org")
	if err != nil {
		t.Fatalf("ListOrgRepos() error = %v", err)
	}
	if len(repos) != 2 || repos[0].Name != "repo-one" || repos[1].Private != true {
		t.Fatalf("ListOrgRepos() = %+v, want 2 repos with repo-one first and repo-two private", repos)
	}
	if repos[0].Owner != "my-org" {
		t.Errorf("repos[0].Owner = %q, want my-org", repos[0].Owner)
	}
}

func TestListContributorStats_RetriesOn202ThenSucceeds(t *testing.T) {
	statsRetryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	t.Cleanup(func() {
		statsRetryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	})

	requests := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/my-org/my-repo/stats/contributors", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests < 2 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		fmt.Fprint(w, `[{"author":{"login":"alice"},"total":5,"weeks":[{"w":1700000000,"a":10,"d":2,"c":3}]}]`)
	})

	c := newTestClient(t, mux)

	stats, err := c.ListContributorStats(context.Background(), "my-org", "my-repo")
	if err != nil {
		t.Fatalf("ListContributorStats() error = %v", err)
	}
	if len(stats) != 1 || stats[0].Login != "alice" {
		t.Fatalf("ListContributorStats() = %+v, want one contributor alice", stats)
	}
	if len(stats[0].Weeks) != 1 || stats[0].Weeks[0].Commits != 3 {
		t.Fatalf("ListContributorStats() weeks = %+v, want one week with 3 commits", stats[0].Weeks)
	}
	if requests != 2 {
		t.Errorf("requests = %d, want 2 (one 202 then one 200)", requests)
	}
}

func TestListContributorStats_PendingAfterBudgetExhausted(t *testing.T) {
	statsRetryDelays = []time.Duration{time.Millisecond}
	t.Cleanup(func() {
		statsRetryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/my-org/my-repo/stats/contributors", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	c := newTestClient(t, mux)

	_, err := c.ListContributorStats(context.Background(), "my-org", "my-repo")
	if err != ErrStatsPending {
		t.Fatalf("ListContributorStats() error = %v, want ErrStatsPending", err)
	}
}
