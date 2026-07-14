// Package httpserver wires the wizard flow (org select -> repo select ->
// leaderboard) to HTTP routes and renders server-side html/template pages.
package httpserver

import (
	"io/fs"
	"net/http"

	"github.com/jaimekb/org-leaderboard/internal/config"
	"github.com/jaimekb/org-leaderboard/internal/githubapi"
	"github.com/jaimekb/org-leaderboard/internal/metrics"
	"github.com/jaimekb/org-leaderboard/internal/wizard"
	"github.com/jaimekb/org-leaderboard/web"
)

// Server holds the app's dependencies and the wired http.Handler.
type Server struct {
	templates   templateSet
	client      githubapi.Client
	store       wizard.Store
	labelConfig *config.LabelConfig
	concurrency int
	window      metrics.TimeWindow

	handler http.Handler
}

// Options configures a new Server.
type Options struct {
	Client      githubapi.Client
	Store       wizard.Store
	LabelConfig *config.LabelConfig
	// Concurrency bounds simultaneous GitHub contributor-stats calls.
	Concurrency int
	// WindowMonths is how far back leaderboard metrics look; defaults to 3.
	WindowMonths int
}

// New builds a Server with routes wired and templates parsed.
func New(opts Options) (*Server, error) {
	ts, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	windowMonths := opts.WindowMonths
	if windowMonths <= 0 {
		windowMonths = 3
	}

	s := &Server{
		templates:   ts,
		client:      opts.Client,
		store:       opts.Store,
		labelConfig: opts.LabelConfig,
		concurrency: opts.Concurrency,
		window:      metrics.LastNMonths(windowMonths),
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/orgs", http.StatusSeeOther)
	})
	mux.HandleFunc("GET /orgs", s.ShowOrgs)
	mux.HandleFunc("POST /orgs", s.SubmitOrgs)
	mux.HandleFunc("GET /repos", s.ShowRepos)
	mux.HandleFunc("POST /repos", s.SubmitRepos)
	mux.HandleFunc("GET /leaderboard/run", s.ShowLeaderboardRun)
	mux.HandleFunc("GET /leaderboard/status", s.LeaderboardStatus)
	mux.HandleFunc("GET /leaderboard", s.ShowLeaderboard)

	if staticFS, err := fs.Sub(web.StaticFS, "static"); err == nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))
	}

	s.handler = withSession(s.store, mux)
}

// Handler returns the fully-wired http.Handler for the app.
func (s *Server) Handler() http.Handler { return s.handler }
