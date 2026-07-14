package httpserver

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jaimekb/org-leaderboard/internal/config"
	"github.com/jaimekb/org-leaderboard/internal/githubapi"
	"github.com/jaimekb/org-leaderboard/internal/wizard"
)

type fakeClient struct {
	orgs  []githubapi.Org
	repos map[string][]githubapi.Repo
	stats map[string][]githubapi.ContributorStats
}

func (f *fakeClient) ListUserOrgs(ctx context.Context) ([]githubapi.Org, error) {
	return f.orgs, nil
}

func (f *fakeClient) ListOrgRepos(ctx context.Context, org string) ([]githubapi.Repo, error) {
	return f.repos[org], nil
}

func (f *fakeClient) ListContributorStats(ctx context.Context, owner, repo string) ([]githubapi.ContributorStats, error) {
	return f.stats[owner+"/"+repo], nil
}

func newTestServer(t *testing.T, client githubapi.Client) *httptest.Server {
	t.Helper()
	s, err := New(Options{
		Client:      client,
		Store:       wizard.NewMemoryStore(),
		LabelConfig: &config.LabelConfig{Version: 1, Orgs: map[string]config.OrgLabels{}},
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv
}

func newTestHTTPClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error = %v", err)
	}
	return &http.Client{Jar: jar}
}

func TestFullWizardFlow_ProducesLeaderboard(t *testing.T) {
	client := &fakeClient{
		orgs: []githubapi.Org{{Login: "acme", ID: 1}},
		repos: map[string][]githubapi.Repo{
			"acme": {{Owner: "acme", Name: "repo1", FullName: "acme/repo1"}},
		},
		stats: map[string][]githubapi.ContributorStats{
			"acme/repo1": {
				{Login: "alice", Weeks: []githubapi.WeeklyStat{{WeekStart: time.Now().Add(-24 * time.Hour), Commits: 7}}},
			},
		},
	}
	srv := newTestServer(t, client)
	httpClient := newTestHTTPClient(t)

	resp, err := httpClient.Post(srv.URL+"/orgs", "application/x-www-form-urlencoded", strings.NewReader(url.Values{"org": {"acme"}}.Encode()))
	if err != nil {
		t.Fatalf("POST /orgs error = %v", err)
	}
	reposBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.Request.URL.Path != "/repos" {
		t.Fatalf("after POST /orgs, path = %s, want /repos", resp.Request.URL.Path)
	}
	if !strings.Contains(string(reposBody), "repo1") {
		t.Fatalf("/repos body does not mention repo1: %s", reposBody)
	}

	resp, err = httpClient.Post(srv.URL+"/repos", "application/x-www-form-urlencoded", strings.NewReader(url.Values{"repo": {"acme|repo1"}}.Encode()))
	if err != nil {
		t.Fatalf("POST /repos error = %v", err)
	}
	resp.Body.Close()
	if resp.Request.URL.Path != "/leaderboard/run" {
		t.Fatalf("after POST /repos, path = %s, want /leaderboard/run", resp.Request.URL.Path)
	}

	deadline := time.Now().Add(2 * time.Second)
	var statusResp *http.Response
	for time.Now().Before(deadline) {
		statusResp, err = httpClient.Get(srv.URL + "/leaderboard/status")
		if err != nil {
			t.Fatalf("GET /leaderboard/status error = %v", err)
		}
		redirect := statusResp.Header.Get("HX-Redirect")
		statusResp.Body.Close()
		if redirect == "/leaderboard" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if statusResp.Header.Get("HX-Redirect") != "/leaderboard" {
		t.Fatalf("pipeline did not finish within deadline")
	}

	resp, err = httpClient.Get(srv.URL + "/leaderboard")
	if err != nil {
		t.Fatalf("GET /leaderboard error = %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if !strings.Contains(string(body), "alice") {
		t.Errorf("/leaderboard body does not mention alice: %s", body)
	}
	if !strings.Contains(string(body), ">7<") {
		t.Errorf("/leaderboard body does not show commit count 7: %s", body)
	}
}

func TestSubmitOrgs_NoneSelectedRedirectsWithError(t *testing.T) {
	client := &fakeClient{orgs: []githubapi.Org{{Login: "acme"}}}
	srv := newTestServer(t, client)
	httpClient := newTestHTTPClient(t)

	resp, err := httpClient.Post(srv.URL+"/orgs", "application/x-www-form-urlencoded", strings.NewReader(""))
	if err != nil {
		t.Fatalf("POST /orgs error = %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.Request.URL.Path != "/orgs" {
		t.Fatalf("path = %s, want /orgs", resp.Request.URL.Path)
	}
	if !strings.Contains(string(body), "select at least one organization") {
		t.Errorf("body does not contain validation error: %s", body)
	}
}

func TestShowRepos_RedirectsToOrgsWhenNoOrgsSelectedYet(t *testing.T) {
	client := &fakeClient{}
	srv := newTestServer(t, client)
	httpClient := newTestHTTPClient(t)

	resp, err := httpClient.Get(srv.URL + "/repos")
	if err != nil {
		t.Fatalf("GET /repos error = %v", err)
	}
	resp.Body.Close()

	if resp.Request.URL.Path != "/orgs" {
		t.Fatalf("path = %s, want /orgs (step guard should redirect back)", resp.Request.URL.Path)
	}
}
