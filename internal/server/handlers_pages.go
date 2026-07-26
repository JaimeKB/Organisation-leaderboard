package server

import (
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/jaimekershawbrown/organisation-leaderboard/internal/config"
	"github.com/jaimekershawbrown/organisation-leaderboard/internal/leaderboard"
)

type indexData struct {
	Orgs []*github.Organization
	Err  string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if !s.hasToken {
		w.WriteHeader(http.StatusServiceUnavailable)
		s.render(w, "index.html", indexData{Err: config.MissingPATMessage})
		return
	}
	orgs, err := s.gh.ListOrgs(r.Context())
	if err != nil {
		s.renderIndexError(w, err)
		return
	}
	s.render(w, "index.html", indexData{Orgs: orgs})
}

func (s *Server) renderIndexError(w http.ResponseWriter, err error) {
	log.Printf("list orgs: %v", err)
	w.WriteHeader(http.StatusBadGateway)
	s.render(w, "index.html", indexData{Err: friendlyGitHubError(err)})
}

type orgDashboardData struct {
	Org          string
	Repos        []*github.Repository
	Members      []*github.User
	Leaderboard  []leaderboard.Contributor
	SinceDate    string
	UntilDate    string
	SelectedRepo map[string]bool
	SelectedUser map[string]bool
	Err          string
}

func (s *Server) handleOrgDashboard(w http.ResponseWriter, r *http.Request) {
	org := r.PathValue("org")
	ctx := r.Context()

	if !s.hasToken {
		w.WriteHeader(http.StatusServiceUnavailable)
		s.render(w, "org.html", orgDashboardData{Org: org, Err: config.MissingPATMessage})
		return
	}

	repos, err := s.gh.ListReposByOrg(ctx, org)
	if err != nil {
		s.renderOrgError(w, org, err)
		return
	}
	members, err := s.gh.ListOrgMembers(ctx, org)
	if err != nil {
		s.renderOrgError(w, org, err)
		return
	}

	until := time.Now()
	since := until.Add(-leaderboard.DefaultWindow)
	rows, err := leaderboard.Compute(ctx, s.gh, leaderboard.Query{Org: org, Since: since, Until: until})
	if err != nil {
		s.renderOrgError(w, org, err)
		return
	}

	s.render(w, "org.html", orgDashboardData{
		Org:          org,
		Repos:        repos,
		Members:      members,
		Leaderboard:  rows,
		SinceDate: since.Format("2006-01-02"),
		UntilDate: until.Format("2006-01-02"),
	})
}

func (s *Server) renderOrgError(w http.ResponseWriter, org string, err error) {
	log.Printf("org dashboard %s: %v", org, err)
	w.WriteHeader(http.StatusBadGateway)
	s.render(w, "org.html", orgDashboardData{Org: org, Err: friendlyGitHubError(err)})
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

func friendlyGitHubError(err error) string {
	if err == nil {
		return ""
	}
	return "Couldn't reach GitHub: " + err.Error()
}
